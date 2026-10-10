package finance

import (
	"context"
	"uuid"

	"selfemi/internal/platform/provenance"
)

const stubValue = "not tracked yet"

type StubStateSource struct{}

func (StubStateSource) Inputs(_ context.Context, _ uuid.UUID) ([]provenance.Labelled, error) {
	return []provenance.Labelled{
		{Key: "available_savings", Label: "Available savings", Value: stubValue, Source: provenance.Stub},
		{Key: "monthly_saving_capacity", Label: "Monthly saving capacity", Value: stubValue, Source: provenance.Stub},
		{Key: "cash_buffer", Label: "Cash buffer", Value: stubValue, Source: provenance.Stub},
	}, nil
}
