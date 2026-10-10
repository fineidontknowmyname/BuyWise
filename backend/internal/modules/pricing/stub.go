package pricing

import (
	"context"
	"time"

	"selfemi/internal/platform/provenance"
)

const StubUnavailableReason = "no forecast in the MVP"

type StubPriceData struct{}

func (StubPriceData) Series(_ context.Context, _ ProductRef, _, _, _ time.Time) (PriceSeries, error) {
	return PriceSeries{
		Observations:   []Observation{},
		Coverage:       "none",
		DatasetVersion: "stub",
		Source:         provenance.Stub,
	}, nil
}

type StubForecast struct{}

func (StubForecast) Forecast(_ context.Context, _ ProductRef, _ time.Duration, _ time.Time) (Forecast, error) {
	return Forecast{
		Available:         false,
		UnavailableReason: StubUnavailableReason,
		MethodVersion:     "stub",
		DatasetVersion:    "stub",
		Source:            provenance.Stub,
	}, nil
}
