package pricing

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"selfemi/internal/platform/httpjson"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /products", h.createProduct)
	mux.HandleFunc("GET /products", h.listProducts)
	mux.HandleFunc("GET /products/{id}", h.getProduct)
	mux.HandleFunc("PUT /products/{id}", h.updateProduct)
	mux.HandleFunc("DELETE /products/{id}", h.deleteProduct)
	mux.HandleFunc("GET /products/{id}/prices", h.listPriceObservations)
	mux.HandleFunc("POST /products/{id}/prices", h.addPriceObservation)
	mux.HandleFunc("POST /products/{id}/prices/import", h.importPrices)
	mux.HandleFunc("GET /products/{id}/stats", h.getProductStats)

	// Goal-specific price observations & document uploads
	mux.HandleFunc("GET /goals/{goalId}/prices", h.listGoalPriceObservations)
	mux.HandleFunc("POST /goals/{goalId}/documents", h.createDocumentUpload)
	mux.HandleFunc("GET /goals/{goalId}/documents", h.listGoalDocumentUploads)
	mux.HandleFunc("GET /goals/{goalId}/documents/{docId}", h.getDocumentUpload)
	mux.HandleFunc("POST /goals/{goalId}/documents/{docId}/confirm", h.confirmDocumentUpload)
}

type productPayload struct {
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
}

type addObservationRequest struct {
	GoalID     *string `json:"goalId,omitempty"`
	PricePaise int64   `json:"pricePaise"`
	Currency   string  `json:"currency"`
	PriceType  string  `json:"priceType"`
	ObservedOn string  `json:"observedOn"`
	Retailer   string  `json:"retailer"`
	Source     string  `json:"source"`
	Provenance string  `json:"provenance"`
	SourceRef  string  `json:"sourceRef"`
	Notes      string  `json:"notes"`
}

type importPricesRequest struct {
	Filename     string                  `json:"filename"`
	Observations []addObservationRequest `json:"observations"`
}

type documentUploadRequest struct {
	ProductID  *string                `json:"productId,omitempty"`
	Filename   string                 `json:"filename"`
	FileType   string                 `json:"fileType"`
	FileSize   int64                  `json:"fileSize"`
	Purpose    string                 `json:"purpose"`
	Status     string                 `json:"status"`
	ParsedData map[string]interface{} `json:"parsedData"`
	Summary    map[string]interface{} `json:"summary"`
}

type confirmDocumentRequest struct {
	Product      *productPayload         `json:"product,omitempty"`
	Observations []addObservationRequest `json:"observations,omitempty"`
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var req productPayload
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid product payload.")
		return
	}
	p := Product{
		Name:               req.Name,
		Brand:              req.Brand,
		Model:              req.Model,
		Variant:            req.Variant,
		Category:           req.Category,
		Description:        req.Description,
		CurrentPricePaise:  req.CurrentPricePaise,
		OriginalPricePaise: req.OriginalPricePaise,
		TargetPricePaise:   req.TargetPricePaise,
		DiscountPercent:    req.DiscountPercent,
		Currency:           req.Currency,
		Availability:       req.Availability,
		Specifications:     req.Specifications,
		URL:                req.URL,
		ImageURL:           req.ImageURL,
		Retailer:           req.Retailer,
		Notes:              req.Notes,
		LastRetrievedAt:    req.LastRetrievedAt,
		Source:             req.Source,
		VerificationStatus: req.VerificationStatus,
		Warnings:           req.Warnings,
	}
	created, err := h.service.CreateProduct(r.Context(), p)
	if errors.Is(err, ErrInvalidName) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_name", "Product name must be 1 to 200 characters.")
		return
	}
	if err != nil {
		h.logger.Error("create product failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create product.")
		return
	}
	httpjson.Write(w, http.StatusCreated, created)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.ListProducts(r.Context())
	if err != nil {
		h.logger.Error("list products failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list products.")
		return
	}
	httpjson.Write(w, http.StatusOK, products)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	prod, err := h.service.GetProduct(r.Context(), id)
	if errors.Is(err, ErrProductNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	if err != nil {
		h.logger.Error("get product failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get product.")
		return
	}
	httpjson.Write(w, http.StatusOK, prod)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	var req productPayload
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid product payload.")
		return
	}
	p := Product{
		ID:                 id,
		Name:               req.Name,
		Brand:              req.Brand,
		Model:              req.Model,
		Variant:            req.Variant,
		Category:           req.Category,
		Description:        req.Description,
		CurrentPricePaise:  req.CurrentPricePaise,
		OriginalPricePaise: req.OriginalPricePaise,
		TargetPricePaise:   req.TargetPricePaise,
		DiscountPercent:    req.DiscountPercent,
		Currency:           req.Currency,
		Availability:       req.Availability,
		Specifications:     req.Specifications,
		URL:                req.URL,
		ImageURL:           req.ImageURL,
		Retailer:           req.Retailer,
		Notes:              req.Notes,
		LastRetrievedAt:    req.LastRetrievedAt,
		Source:             req.Source,
		VerificationStatus: req.VerificationStatus,
		Warnings:           req.Warnings,
	}
	updated, err := h.service.UpdateProduct(r.Context(), p)
	if errors.Is(err, ErrProductNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	if err != nil {
		h.logger.Error("update product failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to modify product.")
		return
	}
	httpjson.Write(w, http.StatusOK, updated)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	if err := h.service.DeleteProduct(r.Context(), id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
			return
		}
		h.logger.Error("delete product failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete product.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listPriceObservations(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	obs, err := h.service.ListPriceObservations(r.Context(), id)
	if err != nil {
		h.logger.Error("list observations failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list price observations.")
		return
	}
	httpjson.Write(w, http.StatusOK, obs)
}

func (h *Handler) addPriceObservation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	var req addObservationRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid observation payload.")
		return
	}
	var goalUUID *uuid.UUID
	if req.GoalID != nil && *req.GoalID != "" {
		if parsed, err := uuid.Parse(*req.GoalID); err == nil {
			goalUUID = &parsed
		}
	}

	obs := PriceObservation{
		ProductID:  id,
		GoalID:     goalUUID,
		PricePaise: req.PricePaise,
		Currency:   req.Currency,
		PriceType:  req.PriceType,
		ObservedOn: req.ObservedOn,
		Retailer:   req.Retailer,
		Source:     req.Source,
		Provenance: req.Provenance,
		SourceRef:  req.SourceRef,
		Notes:      req.Notes,
	}
	created, err := h.service.AddPriceObservation(r.Context(), obs)
	if errors.Is(err, ErrInvalidPrice) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_price", "Price must be greater than zero.")
		return
	}
	if err != nil {
		h.logger.Error("add observation failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to record observation.")
		return
	}
	httpjson.Write(w, http.StatusCreated, created)
}

func (h *Handler) importPrices(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	var req importPricesRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid import payload.")
		return
	}
	var obsList []PriceObservation
	for _, o := range req.Observations {
		var gID *uuid.UUID
		if o.GoalID != nil && *o.GoalID != "" {
			if parsed, err := uuid.Parse(*o.GoalID); err == nil {
				gID = &parsed
			}
		}
		obsList = append(obsList, PriceObservation{
			GoalID:     gID,
			PricePaise: o.PricePaise,
			Currency:   o.Currency,
			PriceType:  o.PriceType,
			ObservedOn: o.ObservedOn,
			Retailer:   o.Retailer,
			Source:     o.Source,
			Provenance: o.Provenance,
			SourceRef:  o.SourceRef,
			Notes:      o.Notes,
		})
	}
	batch, err := h.service.ImportPriceBatch(r.Context(), id, req.Filename, obsList)
	if err != nil {
		h.logger.Error("import prices failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to import price batch.")
		return
	}
	httpjson.Write(w, http.StatusOK, batch)
}

func (h *Handler) getProductStats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	stats, err := h.service.GetProductStats(r.Context(), id)
	if errors.Is(err, ErrProductNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "product_not_found", "Product not found.")
		return
	}
	if err != nil {
		h.logger.Error("get stats failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to calculate product stats.")
		return
	}
	httpjson.Write(w, http.StatusOK, stats)
}

func (h *Handler) listGoalPriceObservations(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(r.PathValue("goalId"))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_goal_id", "Invalid goal ID.")
		return
	}
	obs, err := h.service.ListPriceObservationsForGoal(r.Context(), goalID)
	if err != nil {
		h.logger.Error("list goal price observations failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list goal price observations.")
		return
	}
	httpjson.Write(w, http.StatusOK, obs)
}

func (h *Handler) createDocumentUpload(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(r.PathValue("goalId"))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_goal_id", "Invalid goal ID.")
		return
	}
	var req documentUploadRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid document upload payload.")
		return
	}
	var prodUUID *uuid.UUID
	if req.ProductID != nil && *req.ProductID != "" {
		if parsed, err := uuid.Parse(*req.ProductID); err == nil {
			prodUUID = &parsed
		}
	}

	doc := DocumentUpload{
		GoalID:     goalID,
		ProductID:  prodUUID,
		Filename:   req.Filename,
		FileType:   req.FileType,
		FileSize:   req.FileSize,
		Purpose:    req.Purpose,
		Status:     req.Status,
		ParsedData: req.ParsedData,
		Summary:    req.Summary,
	}

	created, err := h.service.CreateDocumentUpload(r.Context(), doc)
	if err != nil {
		h.logger.Error("create document upload failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to record document upload.")
		return
	}
	httpjson.Write(w, http.StatusCreated, created)
}

func (h *Handler) listGoalDocumentUploads(w http.ResponseWriter, r *http.Request) {
	goalID, err := uuid.Parse(r.PathValue("goalId"))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_goal_id", "Invalid goal ID.")
		return
	}
	docs, err := h.service.ListDocumentUploadsForGoal(r.Context(), goalID)
	if err != nil {
		h.logger.Error("list document uploads failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list document uploads.")
		return
	}
	httpjson.Write(w, http.StatusOK, docs)
}

func (h *Handler) getDocumentUpload(w http.ResponseWriter, r *http.Request) {
	docID, err := uuid.Parse(r.PathValue("docId"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	doc, err := h.service.GetDocumentUpload(r.Context(), docID)
	if errors.Is(err, ErrDocumentNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err != nil {
		h.logger.Error("get document upload failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve document upload.")
		return
	}
	httpjson.Write(w, http.StatusOK, doc)
}

func (h *Handler) confirmDocumentUpload(w http.ResponseWriter, r *http.Request) {
	docID, err := uuid.Parse(r.PathValue("docId"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	var req confirmDocumentRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid confirmation payload.")
		return
	}

	var prod *Product
	if req.Product != nil {
		prod = &Product{
			Name:               req.Product.Name,
			Brand:              req.Product.Brand,
			Model:              req.Product.Model,
			Variant:            req.Product.Variant,
			Category:           req.Product.Category,
			Description:        req.Product.Description,
			CurrentPricePaise:  req.Product.CurrentPricePaise,
			OriginalPricePaise: req.Product.OriginalPricePaise,
			TargetPricePaise:   req.Product.TargetPricePaise,
			DiscountPercent:    req.Product.DiscountPercent,
			Currency:           req.Product.Currency,
			Availability:       req.Product.Availability,
			Specifications:     req.Product.Specifications,
			URL:                req.Product.URL,
			ImageURL:           req.Product.ImageURL,
			Retailer:           req.Product.Retailer,
			Notes:              req.Product.Notes,
			Source:             req.Product.Source,
			VerificationStatus: req.Product.VerificationStatus,
			Warnings:           req.Product.Warnings,
		}
	}

	var obsList []PriceObservation
	for _, o := range req.Observations {
		var gID *uuid.UUID
		if o.GoalID != nil && *o.GoalID != "" {
			if parsed, err := uuid.Parse(*o.GoalID); err == nil {
				gID = &parsed
			}
		}
		obsList = append(obsList, PriceObservation{
			GoalID:     gID,
			PricePaise: o.PricePaise,
			Currency:   o.Currency,
			PriceType:  o.PriceType,
			ObservedOn: o.ObservedOn,
			Retailer:   o.Retailer,
			Source:     o.Source,
			Provenance: o.Provenance,
			SourceRef:  o.SourceRef,
			Notes:      o.Notes,
		})
	}

	confirmed, err := h.service.ConfirmDocumentUpload(r.Context(), docID, prod, obsList)
	if errors.Is(err, ErrDocumentNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err != nil {
		h.logger.Error("confirm document upload failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to confirm document upload.")
		return
	}
	httpjson.Write(w, http.StatusOK, confirmed)
}
