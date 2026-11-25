package pricing

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"selfemi/internal/platform/provenance"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

func (s *Service) CreateProduct(ctx context.Context, p Product) (Product, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return Product{}, ErrInvalidName
	}
	p.Name = name

	now := s.now().UTC()
	var empty uuid.UUID
	if p.ID == empty {
		p.ID = uuid.New()
	}
	p.CreatedAt = now
	p.UpdatedAt = now

	if p.Currency == "" {
		p.Currency = "INR"
	}
	if p.Availability == "" {
		p.Availability = "in_stock"
	}
	if p.Source == "" {
		p.Source = "manual"
	}
	if p.VerificationStatus == "" {
		p.VerificationStatus = "unverified"
	}
	if p.Specifications == nil {
		p.Specifications = make(map[string]string)
	}
	if p.Warnings == nil {
		p.Warnings = make([]string, 0)
	}

	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *Service) GetProduct(ctx context.Context, id uuid.UUID) (Product, error) {
	return s.repo.GetProduct(ctx, id)
}

func (s *Service) ListProducts(ctx context.Context) ([]Product, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) UpdateProduct(ctx context.Context, p Product) (Product, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > 200 {
		return Product{}, ErrInvalidName
	}
	p.Name = name
	p.UpdatedAt = s.now().UTC()

	if p.Currency == "" {
		p.Currency = "INR"
	}
	if p.Availability == "" {
		p.Availability = "in_stock"
	}
	if p.Specifications == nil {
		p.Specifications = make(map[string]string)
	}
	if p.Warnings == nil {
		p.Warnings = make([]string, 0)
	}

	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *Service) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProduct(ctx, id)
}

func (s *Service) AddPriceObservation(ctx context.Context, obs PriceObservation) (PriceObservation, error) {
	if obs.PricePaise <= 0 {
		return PriceObservation{}, ErrInvalidPrice
	}
	if strings.TrimSpace(obs.ObservedOn) == "" {
		return PriceObservation{}, ErrInvalidDate
	}
	var empty uuid.UUID
	if obs.ID == empty {
		obs.ID = uuid.New()
	}
	obs.CreatedAt = s.now().UTC()
	if obs.Currency == "" {
		obs.Currency = "INR"
	}
	if obs.PriceType == "" {
		obs.PriceType = "regular"
	}
	if obs.Source == "" {
		obs.Source = "manual"
	}
	if obs.Provenance == "" {
		obs.Provenance = "manual"
	}

	if err := s.repo.AddPriceObservation(ctx, obs); err != nil {
		return PriceObservation{}, err
	}
	return obs, nil
}

func (s *Service) ListPriceObservations(ctx context.Context, productID uuid.UUID) ([]PriceObservation, error) {
	return s.repo.ListPriceObservations(ctx, productID)
}

func (s *Service) ListPriceObservationsForGoal(ctx context.Context, goalID uuid.UUID) ([]PriceObservation, error) {
	return s.repo.ListPriceObservationsForGoal(ctx, goalID)
}

func (s *Service) ImportPriceBatch(ctx context.Context, productID uuid.UUID, filename string, observations []PriceObservation) (ImportBatch, error) {
	batch := ImportBatch{
		ID:        uuid.New(),
		ProductID: productID,
		Filename:  filename,
		TotalRows: len(observations),
		CreatedAt: s.now().UTC(),
	}

	var validObs []PriceObservation
	errorCount := 0
	now := s.now().UTC()

	for _, obs := range observations {
		if obs.PricePaise <= 0 || strings.TrimSpace(obs.ObservedOn) == "" {
			errorCount++
			continue
		}
		obs.ID = uuid.New()
		obs.ProductID = productID
		obs.CreatedAt = now
		if obs.Currency == "" {
			obs.Currency = "INR"
		}
		if obs.PriceType == "" {
			obs.PriceType = "regular"
		}
		if obs.Source == "" {
			obs.Source = "csv_import"
		}
		if obs.Provenance == "" {
			obs.Provenance = "imported"
		}
		validObs = append(validObs, obs)
	}

	batch.ErrorRows = errorCount
	return s.repo.ImportPriceBatch(ctx, batch, validObs)
}

func (s *Service) GetProductStats(ctx context.Context, productID uuid.UUID) (ProductStats, error) {
	return s.repo.GetProductStats(ctx, productID)
}

func (s *Service) CreateDocumentUpload(ctx context.Context, doc DocumentUpload) (DocumentUpload, error) {
	var empty uuid.UUID
	if doc.ID == empty {
		doc.ID = uuid.New()
	}
	doc.CreatedAt = s.now().UTC()
	if doc.Status == "" {
		doc.Status = "pending"
	}
	if doc.ParsedData == nil {
		doc.ParsedData = make(map[string]interface{})
	}
	if doc.Summary == nil {
		doc.Summary = make(map[string]interface{})
	}

	if err := s.repo.CreateDocumentUpload(ctx, doc); err != nil {
		return DocumentUpload{}, err
	}
	return doc, nil
}

func (s *Service) GetDocumentUpload(ctx context.Context, id uuid.UUID) (DocumentUpload, error) {
	return s.repo.GetDocumentUpload(ctx, id)
}

func (s *Service) ListDocumentUploadsForGoal(ctx context.Context, goalID uuid.UUID) ([]DocumentUpload, error) {
	return s.repo.ListDocumentUploadsForGoal(ctx, goalID)
}

func (s *Service) ConfirmDocumentUpload(ctx context.Context, docID uuid.UUID, confirmedProduct *Product, observations []PriceObservation) (DocumentUpload, error) {
	doc, err := s.repo.GetDocumentUpload(ctx, docID)
	if err != nil {
		return DocumentUpload{}, err
	}

	now := s.now().UTC()
	var productID uuid.UUID

	// If product details were extracted and confirmed, create or update product
	var empty uuid.UUID
	if confirmedProduct != nil {
		if confirmedProduct.ID == empty {
			created, err := s.CreateProduct(ctx, *confirmedProduct)
			if err != nil {
				return DocumentUpload{}, err
			}
			productID = created.ID
		} else {
			updated, err := s.UpdateProduct(ctx, *confirmedProduct)
			if err != nil {
				return DocumentUpload{}, err
			}
			productID = updated.ID
		}
		doc.ProductID = &productID
	} else if doc.ProductID != nil {
		productID = *doc.ProductID
	}

	// Persist price observations if present
	importedCount := 0
	if productID != empty && len(observations) > 0 {
		batch, err := s.ImportPriceBatch(ctx, productID, doc.Filename, observations)
		if err == nil {
			importedCount = batch.ImportedRows
		}
	}

	doc.Status = "confirmed"
	doc.ConfirmedAt = &now
	if doc.Summary == nil {
		doc.Summary = make(map[string]interface{})
	}
	doc.Summary["importedObservations"] = importedCount
	doc.Summary["confirmedAt"] = now.Format(time.RFC3339)

	if err := s.repo.UpdateDocumentUpload(ctx, doc); err != nil {
		return DocumentUpload{}, err
	}
	return doc, nil
}

// Implement PriceDataProvider
func (s *Service) Series(ctx context.Context, ref ProductRef, from, to, asOf time.Time) (PriceSeries, error) {
	if ref == "" {
		return PriceSeries{
			Observations:   []Observation{},
			Coverage:       "none",
			DatasetVersion: "v1",
			Source:         provenance.Stub,
		}, nil
	}
	id, err := uuid.Parse(string(ref))
	if err != nil {
		return PriceSeries{}, err
	}
	obs, err := s.repo.ListPriceObservations(ctx, id)
	if err != nil {
		return PriceSeries{}, err
	}
	var res []Observation
	for _, o := range obs {
		t, _ := time.Parse("2006-01-02", o.ObservedOn)
		res = append(res, Observation{
			ObservedAt: t,
			RecordedAt: o.CreatedAt,
			Price:      string(rune(o.PricePaise)),
			Source:     o.Source,
		})
	}
	return PriceSeries{
		Observations:   res,
		Coverage:       "full",
		DatasetVersion: "v1",
		Source:         provenance.Manual,
	}, nil
}
