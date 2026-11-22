package pricing

import (
	"context"
	"time"
	"uuid"

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

type StubRepository struct {
	Products     map[uuid.UUID]Product
	Observations map[uuid.UUID][]PriceObservation
	Documents    map[uuid.UUID]DocumentUpload
}

func NewStubRepository() *StubRepository {
	return &StubRepository{
		Products:     make(map[uuid.UUID]Product),
		Observations: make(map[uuid.UUID][]PriceObservation),
		Documents:    make(map[uuid.UUID]DocumentUpload),
	}
}

func (s *StubRepository) CreateProduct(_ context.Context, p Product) error {
	s.Products[p.ID] = p
	return nil
}

func (s *StubRepository) GetProduct(_ context.Context, id uuid.UUID) (Product, error) {
	p, ok := s.Products[id]
	if !ok {
		return Product{}, ErrProductNotFound
	}
	return p, nil
}

func (s *StubRepository) ListProducts(_ context.Context) ([]Product, error) {
	var list []Product
	for _, p := range s.Products {
		list = append(list, p)
	}
	return list, nil
}

func (s *StubRepository) UpdateProduct(_ context.Context, p Product) error {
	if _, ok := s.Products[p.ID]; !ok {
		return ErrProductNotFound
	}
	s.Products[p.ID] = p
	return nil
}

func (s *StubRepository) DeleteProduct(_ context.Context, id uuid.UUID) error {
	if _, ok := s.Products[id]; !ok {
		return ErrProductNotFound
	}
	delete(s.Products, id)
	delete(s.Observations, id)
	return nil
}

func (s *StubRepository) AddPriceObservation(_ context.Context, obs PriceObservation) error {
	s.Observations[obs.ProductID] = append(s.Observations[obs.ProductID], obs)
	return nil
}

func (s *StubRepository) ListPriceObservations(_ context.Context, productID uuid.UUID) ([]PriceObservation, error) {
	return s.Observations[productID], nil
}

func (s *StubRepository) ListPriceObservationsForGoal(_ context.Context, goalID uuid.UUID) ([]PriceObservation, error) {
	var out []PriceObservation
	for _, list := range s.Observations {
		for _, obs := range list {
			if obs.GoalID != nil && *obs.GoalID == goalID {
				out = append(out, obs)
			}
		}
	}
	return out, nil
}

func (s *StubRepository) ImportPriceBatch(_ context.Context, batch ImportBatch, observations []PriceObservation) (ImportBatch, error) {
	s.Observations[batch.ProductID] = append(s.Observations[batch.ProductID], observations...)
	batch.ImportedRows = len(observations)
	return batch, nil
}

func (s *StubRepository) GetProductStats(_ context.Context, productID uuid.UUID) (ProductStats, error) {
	p, ok := s.Products[productID]
	if !ok {
		return ProductStats{}, ErrProductNotFound
	}
	return ProductStats{
		ProductID:         productID,
		CurrentPricePaise: p.CurrentPricePaise,
	}, nil
}

func (s *StubRepository) CreateDocumentUpload(_ context.Context, doc DocumentUpload) error {
	s.Documents[doc.ID] = doc
	return nil
}

func (s *StubRepository) GetDocumentUpload(_ context.Context, id uuid.UUID) (DocumentUpload, error) {
	doc, ok := s.Documents[id]
	if !ok {
		return DocumentUpload{}, ErrDocumentNotFound
	}
	return doc, nil
}

func (s *StubRepository) ListDocumentUploadsForGoal(_ context.Context, goalID uuid.UUID) ([]DocumentUpload, error) {
	var out []DocumentUpload
	for _, doc := range s.Documents {
		if doc.GoalID == goalID {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (s *StubRepository) UpdateDocumentUpload(_ context.Context, doc DocumentUpload) error {
	if _, ok := s.Documents[doc.ID]; !ok {
		return ErrDocumentNotFound
	}
	s.Documents[doc.ID] = doc
	return nil
}
