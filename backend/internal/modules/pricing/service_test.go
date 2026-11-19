package pricing

import (
	"context"
	"testing"
	"uuid"
)

type memoryPricingRepo struct {
	products     map[uuid.UUID]Product
	observations map[uuid.UUID][]PriceObservation
	documents    map[uuid.UUID]DocumentUpload
}

func newMemoryPricingRepo() *memoryPricingRepo {
	return &memoryPricingRepo{
		products:     make(map[uuid.UUID]Product),
		observations: make(map[uuid.UUID][]PriceObservation),
		documents:    make(map[uuid.UUID]DocumentUpload),
	}
}

func (m *memoryPricingRepo) CreateProduct(_ context.Context, p Product) error {
	m.products[p.ID] = p
	return nil
}

func (m *memoryPricingRepo) GetProduct(_ context.Context, id uuid.UUID) (Product, error) {
	p, ok := m.products[id]
	if !ok {
		return Product{}, ErrProductNotFound
	}
	return p, nil
}

func (m *memoryPricingRepo) ListProducts(_ context.Context) ([]Product, error) {
	out := make([]Product, 0, len(m.products))
	for _, p := range m.products {
		out = append(out, p)
	}
	return out, nil
}

func (m *memoryPricingRepo) UpdateProduct(_ context.Context, p Product) error {
	if _, ok := m.products[p.ID]; !ok {
		return ErrProductNotFound
	}
	m.products[p.ID] = p
	return nil
}

func (m *memoryPricingRepo) DeleteProduct(_ context.Context, id uuid.UUID) error {
	if _, ok := m.products[id]; !ok {
		return ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

func (m *memoryPricingRepo) AddPriceObservation(_ context.Context, obs PriceObservation) error {
	list := m.observations[obs.ProductID]
	list = append(list, obs)
	m.observations[obs.ProductID] = list
	return nil
}

func (m *memoryPricingRepo) ListPriceObservations(_ context.Context, productID uuid.UUID) ([]PriceObservation, error) {
	return m.observations[productID], nil
}

func (m *memoryPricingRepo) ListPriceObservationsForGoal(_ context.Context, goalID uuid.UUID) ([]PriceObservation, error) {
	var out []PriceObservation
	for _, list := range m.observations {
		for _, obs := range list {
			if obs.GoalID != nil && *obs.GoalID == goalID {
				out = append(out, obs)
			}
		}
	}
	return out, nil
}

func (m *memoryPricingRepo) ImportPriceBatch(_ context.Context, batch ImportBatch, observations []PriceObservation) (ImportBatch, error) {
	batch.ImportedRows = len(observations)
	for _, obs := range observations {
		_ = m.AddPriceObservation(context.Background(), obs)
	}
	return batch, nil
}

func (m *memoryPricingRepo) GetProductStats(_ context.Context, productID uuid.UUID) (ProductStats, error) {
	prod, ok := m.products[productID]
	if !ok {
		return ProductStats{}, ErrProductNotFound
	}
	obsList := m.observations[productID]
	var minP, maxP, sumP int64
	for i, o := range obsList {
		if i == 0 || o.PricePaise < minP {
			minP = o.PricePaise
		}
		if i == 0 || o.PricePaise > maxP {
			maxP = o.PricePaise
		}
		sumP += o.PricePaise
	}
	var avgP int64
	if len(obsList) > 0 {
		avgP = sumP / int64(len(obsList))
	}
	return ProductStats{
		ProductID:         productID,
		CurrentPricePaise: prod.CurrentPricePaise,
		LowestPricePaise:  minP,
		HighestPricePaise: maxP,
		AveragePricePaise: avgP,
		ObservationCount:  len(obsList),
	}, nil
}

func (m *memoryPricingRepo) CreateDocumentUpload(_ context.Context, doc DocumentUpload) error {
	m.documents[doc.ID] = doc
	return nil
}

func (m *memoryPricingRepo) GetDocumentUpload(_ context.Context, id uuid.UUID) (DocumentUpload, error) {
	doc, ok := m.documents[id]
	if !ok {
		return DocumentUpload{}, ErrDocumentNotFound
	}
	return doc, nil
}

func (m *memoryPricingRepo) ListDocumentUploadsForGoal(_ context.Context, goalID uuid.UUID) ([]DocumentUpload, error) {
	var out []DocumentUpload
	for _, doc := range m.documents {
		if doc.GoalID == goalID {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (m *memoryPricingRepo) UpdateDocumentUpload(_ context.Context, doc DocumentUpload) error {
	if _, ok := m.documents[doc.ID]; !ok {
		return ErrDocumentNotFound
	}
	m.documents[doc.ID] = doc
	return nil
}

func TestPricingServiceExtendedProductProfile(t *testing.T) {
	repo := newMemoryPricingRepo()
	svc := NewService(repo, fixedNow)
	ctx := context.Background()

	p := Product{
		Name:               "Sony WH-1000XM5",
		Brand:              "Sony",
		Model:              "WH-1000XM5",
		Variant:            "Silver",
		Description:        "Premium noise cancelling headphones",
		CurrentPricePaise:  2999000,
		OriginalPricePaise: 3499000,
		TargetPricePaise:   2500000,
		DiscountPercent:    14.29,
		Currency:           "INR",
		Availability:       "in_stock",
		Specifications: map[string]string{
			"Battery Life": "30 hours",
			"Bluetooth":    "5.2",
		},
		Retailer:           "Amazon",
		Source:             "url_extraction",
		VerificationStatus: "verified",
	}

	created, err := svc.CreateProduct(ctx, p)
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	var empty uuid.UUID
	if created.ID == empty {
		t.Fatal("expected product ID to be generated")
	}
	if created.Specifications["Battery Life"] != "30 hours" {
		t.Errorf("specs mismatch: %v", created.Specifications)
	}

	// Update product
	created.CurrentPricePaise = 2899000
	updated, err := svc.UpdateProduct(ctx, created)
	if err != nil {
		t.Fatalf("UpdateProduct failed: %v", err)
	}
	if updated.CurrentPricePaise != 2899000 {
		t.Errorf("expected updated price 2899000, got %d", updated.CurrentPricePaise)
	}
}

func TestPricingServicePriceObservationsAndStats(t *testing.T) {
	repo := newMemoryPricingRepo()
	svc := NewService(repo, fixedNow)
	ctx := context.Background()

	prod, _ := svc.CreateProduct(ctx, Product{Name: "MacBook Air M3"})

	goalID := uuid.New()
	obs1 := PriceObservation{
		ProductID:  prod.ID,
		GoalID:     &goalID,
		PricePaise: 11490000,
		ObservedOn: "2026-10-01",
		Source:     "manual",
	}
	obs2 := PriceObservation{
		ProductID:  prod.ID,
		GoalID:     &goalID,
		PricePaise: 10990000,
		ObservedOn: "2026-10-15",
		Source:     "url_extraction",
	}

	if _, err := svc.AddPriceObservation(ctx, obs1); err != nil {
		t.Fatalf("AddPriceObservation 1 failed: %v", err)
	}
	if _, err := svc.AddPriceObservation(ctx, obs2); err != nil {
		t.Fatalf("AddPriceObservation 2 failed: %v", err)
	}

	obsList, err := svc.ListPriceObservations(ctx, prod.ID)
	if err != nil || len(obsList) != 2 {
		t.Fatalf("expected 2 observations, got %v", obsList)
	}

	goalObsList, err := svc.ListPriceObservationsForGoal(ctx, goalID)
	if err != nil || len(goalObsList) != 2 {
		t.Fatalf("expected 2 goal observations, got %v", goalObsList)
	}

	stats, err := svc.GetProductStats(ctx, prod.ID)
	if err != nil {
		t.Fatalf("GetProductStats failed: %v", err)
	}
	if stats.LowestPricePaise != 10990000 || stats.HighestPricePaise != 11490000 {
		t.Errorf("unexpected stats: %+v", stats)
	}
}

func TestPricingServiceDocumentUploadLifecycle(t *testing.T) {
	repo := newMemoryPricingRepo()
	svc := NewService(repo, fixedNow)
	ctx := context.Background()

	goalID := uuid.New()
	doc := DocumentUpload{
		GoalID:   goalID,
		Filename: "sony_xm5_price_history.csv",
		FileType: "csv",
		FileSize: 1024,
		Purpose:  "prices",
		ParsedData: map[string]interface{}{
			"detectedRows": 3,
		},
	}

	createdDoc, err := svc.CreateDocumentUpload(ctx, doc)
	if err != nil {
		t.Fatalf("CreateDocumentUpload failed: %v", err)
	}
	if createdDoc.Status != "pending" {
		t.Errorf("expected pending status, got %s", createdDoc.Status)
	}

	// Confirm document upload with confirmed product & observations
	confirmedProduct := &Product{
		Name:              "Sony WH-1000XM5",
		Brand:             "Sony",
		CurrentPricePaise: 2999000,
	}
	obs := []PriceObservation{
		{
			PricePaise: 2999000,
			ObservedOn: "2026-10-20",
		},
	}

	confirmedDoc, err := svc.ConfirmDocumentUpload(ctx, createdDoc.ID, confirmedProduct, obs)
	if err != nil {
		t.Fatalf("ConfirmDocumentUpload failed: %v", err)
	}
	if confirmedDoc.Status != "confirmed" {
		t.Errorf("expected confirmed status, got %s", confirmedDoc.Status)
	}
	if confirmedDoc.ProductID == nil {
		t.Errorf("expected product ID to be attached to confirmed document")
	}
}
