package pricing

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"selfemi/internal/platform/httpjson"
)

type offerHandler struct {
	store    *Store
	logger   *slog.Logger
	now      func() time.Time
	provider PriceProvider
}

type productOffers struct {
	Product          Product         `json:"product"`
	Offers           []retailerOffer `json:"offers"`
	Thresholds       priceThresholds `json:"thresholds"`
	RefreshAvailable bool            `json:"refreshAvailable"`
}

func (h offerHandler) register(mux *http.ServeMux) {
	mux.HandleFunc("POST /products/validate-listing", h.validateURL)
	mux.HandleFunc("POST /products/tracked", h.createTracked)
	mux.HandleFunc("GET /products/{id}/offers", h.product(h.list))
	mux.HandleFunc("POST /products/{id}/offers", h.product(h.create))
	mux.HandleFunc("PUT /products/{id}/thresholds", h.product(h.thresholds))
	mux.HandleFunc("GET /products/{id}/offers/{offerId}/history", h.offer(h.history))
	mux.HandleFunc("POST /products/{id}/offers/{offerId}/observations", h.offer(h.record))
	mux.HandleFunc("POST /products/{id}/offers/{offerId}/import", h.offer(h.importCSV))
	mux.HandleFunc("POST /products/{id}/offers/{offerId}/refresh", h.offer(h.refresh))
}

func (h offerHandler) createTracked(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name         string              `json:"name"`
		Brand        string              `json:"brand"`
		Model        string              `json:"model"`
		Variant      string              `json:"variant"`
		Category     string              `json:"category"`
		Notes        string              `json:"notes"`
		Offer        *offerInput         `json:"offer"`
		MaximumPaise *int64              `json:"maximumPaise"`
		OpeningPrice *ObservedOfferPrice `json:"openingPrice"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		h.fail(w, errInvalidOffer)
		return
	}
	input.Variant = strings.TrimSpace(input.Variant)
	if thresholds := (priceThresholds{MaximumPaise: input.MaximumPaise}); !thresholds.valid() {
		h.fail(w, errInvalidThresholds)
		return
	}
	if input.Offer != nil {
		if err := input.Offer.validate(input.Variant); err != nil {
			h.fail(w, err)
			return
		}
	}
	now := h.now()
	if input.OpeningPrice != nil {
		if input.Offer == nil || input.OpeningPrice.validate(now) != nil {
			h.fail(w, errInvalidObservation)
			return
		}
	}
	if utf8.RuneCountInString(input.Brand) > 100 || utf8.RuneCountInString(input.Model) > 100 ||
		utf8.RuneCountInString(input.Variant) > 100 || utf8.RuneCountInString(input.Category) > 100 ||
		utf8.RuneCountInString(input.Notes) > 1000 {
		h.fail(w, errInvalidOffer)
		return
	}
	product, err := NewService(h.store, h.now).CreateProduct(r.Context(), Product{
		Name: input.Name, Brand: strings.TrimSpace(input.Brand), Model: strings.TrimSpace(input.Model),
		Variant: input.Variant, Category: input.Category, Notes: input.Notes, Availability: "unknown",
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	if input.Offer != nil {
		offer, err := h.store.createOffer(r.Context(), product.ID, *input.Offer)
		if err != nil {
			h.fail(w, err)
			return
		}
		if input.OpeningPrice != nil {
			if _, err := h.store.writeOfferPrices(r.Context(), offer, []ObservedOfferPrice{*input.OpeningPrice}, "manual", nil); err != nil {
				h.fail(w, err)
				return
			}
		}
	}
	if input.MaximumPaise != nil {
		if err := h.store.saveThresholds(r.Context(), product.ID, priceThresholds{MaximumPaise: input.MaximumPaise}); err != nil {
			h.fail(w, err)
			return
		}
	}
	httpjson.Write(w, 201, product)
}

func (h offerHandler) validateURL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		h.fail(w, errUnsupportedListing)
		return
	}
	link, err := validateListing(input.URL)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, link)
}

func (h offerHandler) product(run func(http.ResponseWriter, *http.Request, Product) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			h.fail(w, ErrProductNotFound)
			return
		}
		product, err := h.store.GetProduct(r.Context(), id)
		if err == nil {
			err = run(w, r, product)
		}
		if err != nil {
			h.fail(w, err)
		}
	}
}

func (h offerHandler) offer(run func(http.ResponseWriter, *http.Request, retailerOffer) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		product, productErr := uuid.Parse(r.PathValue("id"))
		id, offerErr := uuid.Parse(r.PathValue("offerId"))
		if productErr != nil || offerErr != nil {
			h.fail(w, errOfferNotFound)
			return
		}
		offer, err := h.store.getOffer(r.Context(), product, id)
		if err == nil {
			err = run(w, r, offer)
		}
		if err != nil {
			h.fail(w, err)
		}
	}
}

func (h offerHandler) list(w http.ResponseWriter, r *http.Request, product Product) error {
	offers, err := h.store.listOffers(r.Context(), product.ID)
	if err != nil {
		return err
	}
	thresholds, err := h.store.getThresholds(r.Context(), product.ID)
	if err != nil {
		return err
	}
	httpjson.Write(w, 200, productOffers{product, offers, thresholds, h.provider != nil})
	return nil
}

func (h offerHandler) create(w http.ResponseWriter, r *http.Request, product Product) error {
	var input offerInput
	if httpjson.Decode(w, r, &input) != nil {
		return errInvalidOffer
	}
	if err := input.validate(strings.TrimSpace(product.Variant)); err != nil {
		return err
	}
	offer, err := h.store.createOffer(r.Context(), product.ID, input)
	if err != nil {
		return err
	}
	httpjson.Write(w, 201, offer)
	return nil
}

func (h offerHandler) thresholds(w http.ResponseWriter, r *http.Request, product Product) error {
	var input priceThresholds
	if httpjson.Decode(w, r, &input) != nil || !input.valid() {
		return errInvalidThresholds
	}
	if err := h.store.saveThresholds(r.Context(), product.ID, input); err != nil {
		return err
	}
	w.WriteHeader(204)
	return nil
}

func (h offerHandler) history(w http.ResponseWriter, r *http.Request, offer retailerOffer) error {
	history, err := h.store.offerHistory(r.Context(), offer, h.now())
	if err != nil {
		return err
	}
	history.RefreshAvailable = refreshSupported(h.provider, offer)
	httpjson.Write(w, 200, history)
	return nil
}

func (h offerHandler) record(w http.ResponseWriter, r *http.Request, offer retailerOffer) error {
	var input ObservedOfferPrice
	if httpjson.Decode(w, r, &input) != nil {
		return errInvalidObservation
	}
	return h.ingest(w, r, offer, []ObservedOfferPrice{input}, "manual", nil)
}

func (h offerHandler) importCSV(w http.ResponseWriter, r *http.Request, offer retailerOffer) error {
	var input struct {
		CSV string `json:"csv"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		return errInvalidObservation
	}
	observations, err := parseOfferCSV(input.CSV)
	if err != nil {
		httpjson.WriteError(w, 400, "invalid_csv", "Use date and price columns in INR and check each row for valid values.")
		return nil
	}
	return h.ingest(w, r, offer, observations, "csv", nil)
}
func (h offerHandler) refresh(w http.ResponseWriter, r *http.Request, offer retailerOffer) error {
	if h.provider == nil {
		return ErrProviderUnavailable
	}
	if _, err := validateListing(offer.ListingURL); err != nil {
		return err
	}
	if !refreshSupported(h.provider, offer) {
		return errProviderUnsupported
	}
	job, err := h.store.queueRefresh(r.Context(), offer)
	if err != nil {
		return err
	}
	httpjson.Write(w, 202, job)
	return nil
}
func (h offerHandler) ingest(w http.ResponseWriter, r *http.Request, offer retailerOffer, input []ObservedOfferPrice, source string, fetched *time.Time) error {
	now := h.now()
	for i := range input {
		if err := input[i].validate(now); err != nil {
			return err
		}
	}
	count, err := h.store.writeOfferPrices(r.Context(), offer, input, source, fetched)
	if err != nil {
		return err
	}
	status := 200
	if count > 0 {
		status = 201
	}
	httpjson.Write(w, status, map[string]int64{"imported": count, "duplicates": int64(len(input)) - count})
	return nil
}

func (h offerHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrProductNotFound), errors.Is(err, errOfferNotFound):
		httpjson.WriteError(w, 404, "offer_not_found", "Product or retailer offer not found.")
	case errors.Is(err, errUnsupportedListing):
		httpjson.WriteError(w, 400, "unsupported_listing", "Enter a supported HTTPS retailer listing or add an offer manually.")
	case errors.Is(err, errInvalidOffer), errors.Is(err, ErrInvalidName):
		httpjson.WriteError(w, 400, "invalid_offer", "Check the retailer, seller, and listing URL.")
	case errors.Is(err, errVariantMismatch):
		httpjson.WriteError(w, 409, "variant_mismatch", "Confirm that the listing matches the product variant.")
	case errors.Is(err, errOfferConflict):
		httpjson.WriteError(w, 409, "offer_conflict", "This listing or manual seller is already tracked. Check the existing product and variant.")
	case errors.Is(err, errInvalidObservation):
		httpjson.WriteError(w, 400, "invalid_observation", "Check the observation date, availability, and price components.")
	case errors.Is(err, errInvalidThresholds):
		httpjson.WriteError(w, 400, "invalid_thresholds", "Use positive prices in ideal, good, maximum order.")
	case errors.Is(err, ErrProviderRateLimited):
		httpjson.WriteError(w, 429, "provider_rate_limited", "The provider is rate limited. Use manual entry or try later.")
	case errors.Is(err, ErrProviderUnavailable):
		httpjson.WriteError(w, 503, "provider_unavailable", "Automatic price refresh is unavailable. Record a price or import a CSV.")
	case errors.Is(err, errProviderUnsupported):
		httpjson.WriteError(w, 400, "provider_unsupported", "Live refresh supports direct Amazon India product URLs without a specific seller.")
	default:
		h.logger.Error("offer request failed", "error", err)
		httpjson.WriteError(w, 500, "internal_error", "Something went wrong.")
	}
}
