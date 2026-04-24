package ranker

import "bookcabin/internal/pkg/domain"

type Weights struct {
	Price    float64
	Duration float64
	Stops    float64
}

func DefaultWeights() Weights {
	return Weights{Price: 0.6, Duration: 0.3, Stops: 0.1}
}

func Rank(flights []domain.Flight, w Weights) {
	if len(flights) == 0 {
		return
	}
	minPrice, maxPrice := flights[0].Price.Amount, flights[0].Price.Amount
	minDur, maxDur := flights[0].Duration.TotalMinutes, flights[0].Duration.TotalMinutes
	maxStops := flights[0].Stops
	for _, f := range flights[1:] {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minDur {
			minDur = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxDur {
			maxDur = f.Duration.TotalMinutes
		}
		if f.Stops > maxStops {
			maxStops = f.Stops
		}
	}

	priceRange := float64(maxPrice - minPrice)
	durRange := float64(maxDur - minDur)
	stopRange := float64(maxStops)
	total := w.Price + w.Duration + w.Stops
	if total == 0 {
		total = 1
	}

	for i := range flights {
		f := &flights[i]
		priceScore := 1.0
		if priceRange > 0 {
			priceScore = 1 - float64(f.Price.Amount-minPrice)/priceRange
		}
		durScore := 1.0
		if durRange > 0 {
			durScore = 1 - float64(f.Duration.TotalMinutes-minDur)/durRange
		}
		stopScore := 1.0
		if stopRange > 0 {
			stopScore = 1 - float64(f.Stops)/stopRange
		}
		f.Price.BestValue = round4((w.Price*priceScore + w.Duration*durScore + w.Stops*stopScore) / total)
	}
}

func round4(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}
