package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"uuid"

	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

func (h offerHandler) refreshDetails(w http.ResponseWriter, r *http.Request, offer retailerOffer) error {
	var input struct {
		Confirmed bool `json:"confirmed"`
	}
	if httpjson.Decode(w, r, &input) != nil || !input.Confirmed {
		return errVariantMismatch
	}
	provider, ok := h.provider.(listingRetriever)
	if !ok {
		return ErrProviderUnavailable
	}
	if amazonListingASIN(offer.ListingURL) == "" || offer.Seller != "" {
		return errProviderUnsupported
	}
	id, err := uuid.Parse(offer.ProductID)
	if err != nil {
		return ErrProductNotFound
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	preview, err := provider.LookupListing(ctx, offer.ListingURL)
	if err != nil {
		return err
	}
	if preview.Name == "" || amazonListingASIN(preview.URL) != amazonListingASIN(offer.ListingURL) {
		return errProviderNoPrice
	}
	product, err := h.store.saveListingDetails(r.Context(), id, preview, h.now())
	if err != nil {
		return err
	}
	if preview.CurrentPricePaise > 0 && preview.ObservedAt != nil && preview.Source != "" {
		stock := preview.Availability
		if stock == "" {
			stock = "unknown"
		}
		observation := ObservedOfferPrice{ObservedOn: marketDay(*preview.ObservedAt), ObservedAt: preview.ObservedAt,
			SourceLabel: preview.Source, ItemPaise: preview.CurrentPricePaise, Availability: stock}
		if observation.validate(h.now()) != nil {
			return errInvalidObservation
		}
		fetched := h.now().UTC()
		if _, err := h.store.writeOfferPrices(r.Context(), offer, []ObservedOfferPrice{observation}, "provider", &fetched); err != nil {
			return err
		}
	}
	httpjson.Write(w, 200, product)
	return nil
}

func (s *Store) saveListingDetails(ctx context.Context, id uuid.UUID, preview listingPreview, now time.Time) (Product, error) {
	if err := s.lockOfferProduct(ctx, id); err != nil {
		return Product{}, err
	}
	warnings, err := json.Marshal(preview.Warnings)
	if err != nil {
		return Product{}, err
	}
	if preview.Warnings == nil {
		warnings = []byte("[]")
	}
	_, err = database.Scoped(ctx, s.pool).Exec(ctx, `UPDATE pricing.products SET
		name = $2, brand = CASE WHEN btrim(brand) = '' THEN $3 ELSE brand END,
		image_url = CASE WHEN $4 <> '' THEN $4 ELSE image_url END,
		category = CASE WHEN btrim(category) = '' THEN $5 ELSE category END,
		last_retrieved_at = $6, warnings = $7::jsonb, updated_at = $8 WHERE id = $1::uuid`,
		id.String(), clippedListingText(preview.Name, 200), clippedListingText(preview.Brand, 100), safeProductImage(preview.ImageURL), clippedListingText(preview.Category, 100), preview.ObservedAt, warnings, now.UTC())
	if err != nil {
		return Product{}, err
	}
	return s.GetProduct(ctx, id)
}
