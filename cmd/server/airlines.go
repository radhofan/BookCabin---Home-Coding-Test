package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"bookcabin/internal/pkg/airlinemock"
	"bookcabin/internal/pkg/providers"
)

// startAirlineServers starts one HTTP server per airline on a random local port,
// reads mock JSON data from test/testdata/, and returns the base URLs for use
// by the provider HTTP clients. The servers run until the process exits.
func startAirlineServers() (providers.URLs, error) {
	garuda, err := os.ReadFile("test/testdata/garuda.json")
	if err != nil {
		return providers.URLs{}, fmt.Errorf("read garuda data: %w", err)
	}
	lion, err := os.ReadFile("test/testdata/lion.json")
	if err != nil {
		return providers.URLs{}, fmt.Errorf("read lion data: %w", err)
	}
	batik, err := os.ReadFile("test/testdata/batik.json")
	if err != nil {
		return providers.URLs{}, fmt.Errorf("read batik data: %w", err)
	}
	airasia, err := os.ReadFile("test/testdata/airasia.json")
	if err != nil {
		return providers.URLs{}, fmt.Errorf("read airasia data: %w", err)
	}

	start := func(name string, handler http.Handler) (string, error) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}
		srv := &http.Server{Handler: handler}
		go func() {
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("airline mock %s: %v", name, err)
			}
		}()
		addr := "http://" + ln.Addr().String()
		log.Printf("airline mock: %-20s %s", name, addr)
		return addr, nil
	}

	var urls providers.URLs
	urls.Garuda, err = start("Garuda Indonesia", airlinemock.GarudaHandler{Data: garuda})
	if err != nil {
		return urls, err
	}
	urls.Lion, err = start("Lion Air", airlinemock.LionHandler{Data: lion})
	if err != nil {
		return urls, err
	}
	urls.Batik, err = start("Batik Air", airlinemock.BatikHandler{Data: batik})
	if err != nil {
		return urls, err
	}
	urls.AirAsia, err = start("AirAsia", airlinemock.AirAsiaHandler{Data: airasia})
	if err != nil {
		return urls, err
	}

	return urls, nil
}
