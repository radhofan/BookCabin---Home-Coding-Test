package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookcabin/internal/app/api"
	"bookcabin/internal/app/service"
	"bookcabin/internal/pkg/aggregator"
	"bookcabin/internal/pkg/cache"
	"bookcabin/internal/pkg/providers"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	cacheTTL := flag.Duration("cache-ttl", 30*time.Second, "search result TTL")
	flag.Parse()

	cfg := aggregator.DefaultConfig()
	agg := aggregator.New(cfg, providers.All())
	c := cache.New(*cacheTTL)
	svc := service.New(agg, c)
	h := api.New(svc)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	idleClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		close(idleClosed)
	}()

	log.Printf("bookcabin listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
	<-idleClosed
}
