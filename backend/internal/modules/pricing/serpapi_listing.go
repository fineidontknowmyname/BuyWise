package pricing

import (
	"context"
	"net/url"
	"strings"
	"time"
)

type cachedListingPreview struct {
	result  listingPreview
	expires time.Time
}

func clippedListingText(value string, limit int) string {
	chars := []rune(strings.Join(strings.Fields(value), " "))
	if len(chars) > limit {
		chars = chars[:limit]
	}
	return string(chars)
}

func (p serpProduct) preview(raw string, now time.Time) (listingPreview, error) {
	asin := amazonListingASIN(raw)
	if asin == "" || p.Metadata.Status != "Success" || !strings.EqualFold(p.Product.ASIN, asin) ||
		(p.Product.Link != "" && amazonListingASIN(p.Product.Link) != asin) || strings.TrimSpace(p.Product.Title) == "" {
		return listingPreview{}, errProviderNoPrice
	}
	stamp, err := time.Parse("2006-01-02 15:04:05 MST", p.Metadata.Created)
	if err != nil {
		stamp, err = time.Parse(time.RFC3339, p.Metadata.Created)
	}
	if err != nil || stamp.Before(now.Add(-24*time.Hour)) || stamp.After(now) {
		return listingPreview{}, errProviderNoPrice
	}
	link, _ := validateListing(raw)
	preview := listingPreview{
		URL: link.URL, Retailer: link.Retailer, Name: clippedListingText(p.Product.Title, 200),
		Brand:    clippedListingText(strings.TrimPrefix(p.Product.Brand, "Brand: "), 100),
		ImageURL: safeProductImage(p.Product.Thumbnail), ObservedAt: &stamp, Source: serpSource,
		Warnings: []string{"Review the product and selected variant before saving. Item prices may exclude delivery, taxes and checkout discounts."},
	}
	var variants []string
	for _, group := range p.Product.Variants {
		for _, item := range group.Items {
			if item.Selected && strings.TrimSpace(item.Name) != "" {
				variants = append(variants, group.Title+": "+item.Name)
				break
			}
		}
	}
	preview.Variant = clippedListingText(strings.Join(variants, " / "), 100)
	if len(p.Product.Categories) > 0 {
		preview.Category = clippedListingText(p.Product.Categories[0].Name, 100)
	}
	if price, err := p.observation(asin, now); err == nil {
		preview.CurrentPricePaise = price.ItemPaise
	} else {
		preview.Warnings = append(preview.Warnings, "No matching INR item price was returned. Enter a price manually if needed.")
	}
	preview.ManualDetailsRequired = preview.Variant == "" || preview.CurrentPricePaise == 0
	return preview, nil
}

func (s *SerpAPI) LookupListing(ctx context.Context, raw string) (listingPreview, error) {
	asin := amazonListingASIN(raw)
	if asin == "" {
		return listingPreview{}, errProviderUnsupported
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	now := time.Now().UTC()
	if cached, found := s.previews[asin]; found && cached.expires.After(now) {
		preview := cached.result
		link, _ := validateListing(raw)
		preview.URL = link.URL
		return preview, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return listingPreview{}, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
		return listingPreview{}, err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('buywise.serpapi', 0))"); err != nil {
		return listingPreview{}, err
	}
	if err := s.reserve(ctx, tx, false); err != nil {
		return listingPreview{}, err
	}
	var result serpProduct
	lookupErr := s.get(ctx, "/search.json", url.Values{"engine": {"amazon_product"}, "amazon_domain": {"amazon.in"}, "asin": {asin}}, &result)
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	preview := listingPreview{}
	if lookupErr == nil {
		preview, lookupErr = result.preview(raw, time.Now().UTC())
	}
	if lookupErr == nil {
		if price, priceErr := result.observation(asin, time.Now().UTC()); priceErr == nil {
			if _, err := tx.Exec(saveCtx, `INSERT INTO pricing.serpapi_observation_cache (asin, observed_at, item_paise, availability)
				VALUES ($1, $2, $3, $4) ON CONFLICT (asin) DO UPDATE SET observed_at = EXCLUDED.observed_at,
				item_paise = EXCLUDED.item_paise, availability = EXCLUDED.availability`, asin, price.ObservedAt, price.ItemPaise, price.Availability); err != nil {
				return listingPreview{}, err
			}
		}
	}
	if err := tx.Commit(saveCtx); err != nil {
		return listingPreview{}, err
	}
	if lookupErr != nil {
		return listingPreview{}, lookupErr
	}
	if s.previews == nil {
		s.previews = make(map[string]cachedListingPreview)
	}
	for key, cached := range s.previews {
		if !cached.expires.After(now) {
			delete(s.previews, key)
		}
	}
	if len(s.previews) >= 128 {
		for key := range s.previews {
			delete(s.previews, key)
			break
		}
	}
	s.previews[asin] = cachedListingPreview{result: preview, expires: now.Add(time.Hour)}
	return preview, nil
}
