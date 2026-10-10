package pricing

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"selfemi/internal/platform/provenance"
)

var fixedNow = func() time.Time { return time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC) }

func TestStubPriceDataContract(t *testing.T) {
	series, err := StubPriceData{}.Series(context.Background(), "", time.Time{}, fixedNow(), fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if series.Source != provenance.Stub || series.Observations == nil || len(series.Observations) != 0 {
		t.Fatalf("series = %+v", series)
	}
}

func TestStubForecastIsUnavailable(t *testing.T) {
	f, err := StubForecast{}.Forecast(context.Background(), "", time.Hour, fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	if f.Available || f.UnavailableReason == "" || f.Source != provenance.Stub {
		t.Fatalf("forecast = %+v", f)
	}
}

func TestDecisionInputsWithStubs(t *testing.T) {
	inputs, err := NewDecisionInputs(StubPriceData{}, StubForecast{}, fixedNow).Inputs(context.Background(), uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 {
		t.Fatalf("inputs = %+v", inputs)
	}
	if inputs[0].Key != "price_history" || inputs[0].Value != "0 observations" || inputs[0].Source != provenance.Stub {
		t.Errorf("price input = %+v", inputs[0])
	}
	if inputs[1].Key != "price_forecast" || inputs[1].Value != "unavailable: "+StubUnavailableReason || inputs[1].Source != provenance.Stub {
		t.Errorf("forecast input = %+v", inputs[1])
	}
}

type recordingPrices struct {
	from, to, asOf time.Time
}

func (r *recordingPrices) Series(_ context.Context, _ ProductRef, from, to, asOf time.Time) (PriceSeries, error) {
	r.from, r.to, r.asOf = from, to, asOf
	return PriceSeries{Source: provenance.Manual}, nil
}

type availableForecast struct{}

func (availableForecast) Forecast(context.Context, ProductRef, time.Duration, time.Time) (Forecast, error) {
	return Forecast{Available: true, Low: "55000.00", High: "60000.00", Source: provenance.Calculated}, nil
}

func TestDecisionInputsUseAsOfNowAndShowAvailableRange(t *testing.T) {
	prices := &recordingPrices{}
	inputs, err := NewDecisionInputs(prices, availableForecast{}, fixedNow).Inputs(context.Background(), uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	if !prices.asOf.Equal(fixedNow()) || !prices.to.Equal(fixedNow()) || !prices.from.Equal(fixedNow().Add(-365*24*time.Hour)) {
		t.Errorf("window = %v..%v as of %v", prices.from, prices.to, prices.asOf)
	}
	if inputs[1].Value != "55000.00 to 60000.00" || inputs[1].Source != provenance.Calculated {
		t.Errorf("forecast input = %+v", inputs[1])
	}
}

type failingForecast struct{}

func (failingForecast) Forecast(context.Context, ProductRef, time.Duration, time.Time) (Forecast, error) {
	return Forecast{}, errors.New("boom")
}

func TestDecisionInputsReportProviderErrors(t *testing.T) {
	if _, err := NewDecisionInputs(StubPriceData{}, failingForecast{}, fixedNow).Inputs(context.Background(), uuid.NewV7()); err == nil {
		t.Fatal("expected an error")
	}
}
