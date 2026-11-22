package pricing

import (
	"context"
	"errors"
	"time"
	"uuid"
)

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrDocumentNotFound = errors.New("document not found")
	ErrInvalidPrice     = errors.New("price must be greater than zero")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrInvalidName      = errors.New("product name must be 1 to 200 characters")
)

type Product struct {
	ID                 uuid.UUID         `json:"id"`
	Name               string            `json:"name"`
	Brand              string            `json:"brand"`
	Model              string            `json:"model"`
	Variant            string            `json:"variant"`
	Category           string            `json:"category"`
	Description        string            `json:"description"`
	CurrentPricePaise  int64             `json:"currentPricePaise"`
	OriginalPricePaise int64             `json:"originalPricePaise"`
	TargetPricePaise   int64             `json:"targetPricePaise"`
	DiscountPercent    float64           `json:"discountPercent"`
	Currency           string            `json:"currency"`
	Availability       string            `json:"availability"`
	Specifications     map[string]string `json:"specifications"`
	URL                string            `json:"url"`
	ImageURL           string            `json:"imageUrl"`
	Retailer           string            `json:"retailer"`
	Notes              string            `json:"notes"`
	LastRetrievedAt    *time.Time        `json:"lastRetrievedAt,omitempty"`
	Source             string            `json:"source"`
	VerificationStatus string            `json:"verificationStatus"`
	Warnings           []string          `json:"warnings"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type PriceObservation struct {
	ID            uuid.UUID  `json:"id"`
	ProductID     uuid.UUID  `json:"productId"`
	GoalID        *uuid.UUID `json:"goalId,omitempty"`
	PricePaise    int64      `json:"pricePaise"`
	Currency      string     `json:"currency"`
	PriceType     string     `json:"priceType"`
	ObservedOn    string     `json:"observedOn"` // YYYY-MM-DD
	Retailer      string     `json:"retailer"`
	Source        string     `json:"source"`
	Provenance    string     `json:"provenance"`
	SourceRef     string     `json:"sourceRef"`
	Notes         string     `json:"notes"`
	ImportBatchID *uuid.UUID `json:"importBatchId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type ProductStats struct {
	ProductID         uuid.UUID `json:"productId"`
	CurrentPricePaise int64     `json:"currentPricePaise"`
	LowestPricePaise  int64     `json:"lowestPricePaise"`
	HighestPricePaise int64     `json:"highestPricePaise"`
	AveragePricePaise int64     `json:"averagePricePaise"`
	LatestPricePaise  int64     `json:"latestPricePaise"`
	ObservationCount  int       `json:"observationCount"`
	FirstObservedOn   string    `json:"firstObservedOn"`
	LastObservedOn    string    `json:"lastObservedOn"`
}

type ImportBatch struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"productId"`
	Filename      string    `json:"filename"`
	TotalRows     int       `json:"totalRows"`
	ImportedRows  int       `json:"importedRows"`
	DuplicateRows int       `json:"duplicateRows"`
	ErrorRows     int       `json:"errorRows"`
	CreatedAt     time.Time `json:"createdAt"`
}

type DocumentUpload struct {
	ID          uuid.UUID              `json:"id"`
	GoalID      uuid.UUID              `json:"goalId"`
	ProductID   *uuid.UUID             `json:"productId,omitempty"`
	Filename    string                 `json:"filename"`
	FileType    string                 `json:"fileType"`
	FileSize    int64                  `json:"fileSize"`
	Purpose     string                 `json:"purpose"`
	Status      string                 `json:"status"` // pending, previewed, confirmed, failed
	ParsedData  map[string]interface{} `json:"parsedData"`
	Summary     map[string]interface{} `json:"summary"`
	CreatedAt   time.Time              `json:"createdAt"`
	ConfirmedAt *time.Time             `json:"confirmedAt,omitempty"`
}

type Repository interface {
	CreateProduct(ctx context.Context, p Product) error
	GetProduct(ctx context.Context, id uuid.UUID) (Product, error)
	ListProducts(ctx context.Context) ([]Product, error)
	UpdateProduct(ctx context.Context, p Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	AddPriceObservation(ctx context.Context, obs PriceObservation) error
	ListPriceObservations(ctx context.Context, productID uuid.UUID) ([]PriceObservation, error)
	ListPriceObservationsForGoal(ctx context.Context, goalID uuid.UUID) ([]PriceObservation, error)
	ImportPriceBatch(ctx context.Context, batch ImportBatch, observations []PriceObservation) (ImportBatch, error)
	GetProductStats(ctx context.Context, productID uuid.UUID) (ProductStats, error)
	CreateDocumentUpload(ctx context.Context, doc DocumentUpload) error
	GetDocumentUpload(ctx context.Context, id uuid.UUID) (DocumentUpload, error)
	ListDocumentUploadsForGoal(ctx context.Context, goalID uuid.UUID) ([]DocumentUpload, error)
	UpdateDocumentUpload(ctx context.Context, doc DocumentUpload) error
}
