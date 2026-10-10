package pricing

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"selfemi/internal/platform/provenance"
)

type DecisionInputs struct {
	prices    PriceDataProvider
	forecasts ForecastProvider
	now       func() time.Time
	lookback  time.Duration
	horizon   time.Duration
}

func NewDecisionInputs(prices PriceDataProvider, forecasts ForecastProvider, now func() time.Time) *DecisionInputs {
	return &DecisionInputs{
		prices:    prices,
		forecasts: forecasts,
		now:       now,
		lookback:  365 * 24 * time.Hour,
		horizon:   90 * 24 * time.Hour,
	}
}

func (d *DecisionInputs) Inputs(ctx context.Context, _ uuid.UUID) ([]provenance.Labelled, error) {
	asOf := d.now().UTC()
	ref := ProductRef("")

	series, err := d.prices.Series(ctx, ref, asOf.Add(-d.lookback), asOf, asOf)
	if err != nil {
		return nil, fmt.Errorf("price series: %w", err)
	}
	forecast, err := d.forecasts.Forecast(ctx, ref, d.horizon, asOf)
	if err != nil {
		return nil, fmt.Errorf("forecast: %w", err)
	}

	priceValue := fmt.Sprintf("%d observations", len(series.Observations))
	forecastValue := "unavailable: " + forecast.UnavailableReason
	if forecast.Available {
		forecastValue = forecast.Low + " to " + forecast.High
	}

	return []provenance.Labelled{
		{Key: "price_history", Label: "Price history", Value: priceValue, Source: series.Source},
		{Key: "price_forecast", Label: "Price forecast", Value: forecastValue, Source: forecast.Source},
	}, nil
}
