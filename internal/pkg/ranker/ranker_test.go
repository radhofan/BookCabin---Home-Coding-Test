package ranker

import (
	"testing"

	"bookcabin/internal/pkg/domain"
)

func TestRank_AssignsScoreInRange(t *testing.T) {
	flights := []domain.Flight{
		{Price: domain.Price{Amount: 500000}, Duration: domain.Duration{TotalMinutes: 100}, Stops: 0},
		{Price: domain.Price{Amount: 1500000}, Duration: domain.Duration{TotalMinutes: 300}, Stops: 2},
		{Price: domain.Price{Amount: 900000}, Duration: domain.Duration{TotalMinutes: 150}, Stops: 0},
	}
	Rank(flights, DefaultWeights())
	for _, f := range flights {
		if f.Price.BestValue < 0 || f.Price.BestValue > 1 {
			t.Errorf("score out of range: %v", f.Price.BestValue)
		}
	}
	if flights[0].Price.BestValue <= flights[1].Price.BestValue {
		t.Errorf("cheapest should outscore worst: %.3f vs %.3f",
			flights[0].Price.BestValue, flights[1].Price.BestValue)
	}
}

func TestRank_SingleFlight(t *testing.T) {
	flights := []domain.Flight{{Price: domain.Price{Amount: 100}, Duration: domain.Duration{TotalMinutes: 60}}}
	Rank(flights, DefaultWeights())
	if flights[0].Price.BestValue != 1.0 {
		t.Errorf("single flight score = %v, want 1.0", flights[0].Price.BestValue)
	}
}
