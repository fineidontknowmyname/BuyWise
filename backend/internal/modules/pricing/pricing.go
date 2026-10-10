package pricing

import (
	"context"
	"time"

	"selfemi/internal/platform/provenance"
)

const ContractVersion = "v0"

type ProductRef string

type Observation struct {
	ObservedAt time.Time
	RecordedAt time.Time
	Price      string
	Kind       string
	Source     string
	Confidence string
}

type PriceSeries struct {
	Observations   []Observation
	Coverage       string
	DatasetVersion string
	Source         provenance.Source
}

type Forecast struct {
	Available         bool
	UnavailableReason string
	Low               string
	High              string
	Confidence        string
	MethodVersion     string
	DatasetVersion    string
	Source            provenance.Source
}

type PriceDataProvider interface {
	Series(ctx context.Context, ref ProductRef, from, to, asOf time.Time) (PriceSeries, error)
}

type ForecastProvider interface {
	Forecast(ctx context.Context, ref ProductRef, horizon time.Duration, asOf time.Time) (Forecast, error)
}
