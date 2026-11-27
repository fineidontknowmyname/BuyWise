package pricing

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const maxOfferPaise int64 = 1000000000000
const maxHistoryRows = 200
const marketTimeOffset = 19800
const priceFreshnessDays = 2

var errUnsupportedListing = errors.New("unsupported retailer listing")
var errInvalidOffer = errors.New("invalid retailer offer")
var errVariantMismatch = errors.New("product variant needs confirmation")
var errOfferConflict = errors.New("retailer offer already exists")
var errOfferNotFound = errors.New("retailer offer not found")
var errInvalidObservation = errors.New("invalid price observation")
var errInvalidThresholds = errors.New("invalid price thresholds")
var ErrProviderUnavailable = errors.New("price provider unavailable")
var ErrProviderRateLimited = errors.New("price provider rate limited")

var retailerHosts = map[string]string{
	"amazon.in": "Amazon", "flipkart.com": "Flipkart", "croma.com": "Croma",
	"reliancedigital.in": "Reliance Digital", "apple.com": "Apple",
}

type listingLink struct {
	URL      string `json:"url"`
	Retailer string `json:"retailer"`
}

func validateListing(raw string) (listingLink, error) {
	value := strings.TrimSpace(raw)
	u, err := url.Parse(value)
	if err != nil || len(value) > 2000 || u.Scheme != "https" || u.User != nil ||
		u.Port() != "" || u.Opaque != "" || u.Path == "" || strings.Trim(u.Path, "/") == "" {
		return listingLink{}, errUnsupportedListing
	}
	host := strings.ToLower(u.Hostname())
	retailer, ok := retailerHosts[strings.TrimPrefix(host, "www.")]
	if !ok {
		return listingLink{}, errUnsupportedListing
	}
	u.Host = "www." + strings.TrimPrefix(host, "www.")
	u.Fragment = ""
	q := u.Query()
	for k := range q {
		key := strings.ToLower(k)
		if strings.HasPrefix(key, "utm_") || key == "ref" || key == "tag" || key == "gclid" || key == "fbclid" {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	normalized := u.String()
	if len(normalized) > 2000 {
		return listingLink{}, errUnsupportedListing
	}
	return listingLink{normalized, retailer}, nil
}

type offerInput struct {
	Retailer   string `json:"retailer"`
	Seller     string `json:"seller"`
	ListingURL string `json:"listingUrl"`
	Variant    string `json:"variant"`
	Confirmed  bool   `json:"variantConfirmed"`
}

func (i *offerInput) validate(variant string) error {
	i.Retailer = strings.TrimSpace(i.Retailer)
	i.Seller = strings.TrimSpace(i.Seller)
	i.ListingURL = strings.TrimSpace(i.ListingURL)
	if !i.Confirmed || strings.TrimSpace(i.Variant) != variant {
		return errVariantMismatch
	}
	if i.ListingURL != "" {
		link, err := validateListing(i.ListingURL)
		if err != nil {
			return err
		}
		if i.Retailer != "" && !strings.EqualFold(i.Retailer, link.Retailer) {
			return errInvalidOffer
		}
		i.ListingURL, i.Retailer = link.URL, link.Retailer
	}
	if i.Retailer == "" || utf8.RuneCountInString(i.Retailer) > 100 || utf8.RuneCountInString(i.Seller) > 150 {
		return errInvalidOffer
	}
	return nil
}

type retailerOffer struct {
	ID         string `json:"id"`
	ProductID  string `json:"productId"`
	Retailer   string `json:"retailer"`
	Seller     string `json:"seller"`
	ListingURL string `json:"listingUrl"`
}

type ObservedOfferPrice struct {
	ObservedOn    string     `json:"observedOn"`
	ObservedAt    *time.Time `json:"observedAt"`
	SourceLabel   string     `json:"sourceLabel"`
	Availability  string     `json:"availability"`
	ItemPaise     int64      `json:"itemPaise"`
	TaxPaise      *int64     `json:"taxPaise"`
	DeliveryPaise *int64     `json:"deliveryPaise"`
	DiscountPaise *int64     `json:"discountPaise"`
}

type priceRecord struct {
	ObservedOfferPrice
	ID          string     `json:"id"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	RecordedAt  time.Time  `json:"recordedAt"`
	Source      string     `json:"source"`
	LandedPaise *int64     `json:"landedPaise"`
}

func marketDay(now time.Time) string {
	return now.In(time.FixedZone("IST", marketTimeOffset)).Format(time.DateOnly)
}

func (o *ObservedOfferPrice) validate(now time.Time) error {
	o.SourceLabel = strings.TrimSpace(o.SourceLabel)
	if o.Availability == "" {
		o.Availability = "unknown"
	}
	switch o.Availability {
	case "unknown", "in_stock", "out_of_stock", "pre_order":
	default:
		return errInvalidObservation
	}
	date, err := time.Parse(time.DateOnly, o.ObservedOn)
	if err != nil || date.Year() < 1 || o.ObservedOn > marketDay(now) ||
		o.ItemPaise <= 0 || o.ItemPaise > maxOfferPaise || utf8.RuneCountInString(o.SourceLabel) > 150 {
		return errInvalidObservation
	}
	if o.ObservedAt != nil && (o.ObservedAt.After(now) || marketDay(*o.ObservedAt) != o.ObservedOn) {
		return errInvalidObservation
	}
	parts := []*int64{o.TaxPaise, o.DeliveryPaise, o.DiscountPaise}
	for _, part := range parts {
		if part != nil && (*part < 0 || *part > maxOfferPaise) {
			return errInvalidObservation
		}
	}
	if o.DiscountPaise != nil {
		gross := o.ItemPaise
		if o.TaxPaise != nil {
			gross += *o.TaxPaise
		}
		if o.DeliveryPaise != nil {
			gross += *o.DeliveryPaise
		}
		if *o.DiscountPaise >= gross {
			return errInvalidObservation
		}
	}
	return nil
}

type priceThresholds struct {
	IdealPaise   *int64 `json:"idealPaise"`
	GoodPaise    *int64 `json:"goodPaise"`
	MaximumPaise *int64 `json:"maximumPaise"`
}

func (t priceThresholds) valid() bool {
	last := int64(0)
	for _, value := range []*int64{t.IdealPaise, t.GoodPaise, t.MaximumPaise} {
		if value != nil {
			if *value <= 0 || *value > maxOfferPaise || *value < last {
				return false
			}
			last = *value
		}
	}
	return true
}

type priceStats struct {
	Count        int64  `json:"count"`
	MinimumPaise *int64 `json:"minimumPaise"`
	MaximumPaise *int64 `json:"maximumPaise"`
	AveragePaise *int64 `json:"averagePaise"`
}

type offerHistory struct {
	RefreshAvailable bool           `json:"refreshAvailable"`
	Refresh          *refreshStatus `json:"refresh"`
	Offer            retailerOffer  `json:"offer"`
	Observations     []priceRecord  `json:"observations"`
	Stats            priceStats     `json:"stats"`
	Freshness        string         `json:"freshness"`
}

func priceFreshness(latest string, now time.Time) string {
	if latest == "" {
		return "missing"
	}
	if latest < marketDay(now.AddDate(0, 0, -priceFreshnessDays)) {
		return "stale"
	}
	return "recent"
}

type ProviderListing struct {
	Seller    string
	URL       string
	Brand     string
	Model     string
	Variant   string
	Scheduled bool
}

type PriceProvider interface {
	Observe(context.Context, ProviderListing) (ObservedOfferPrice, error)
}

func refreshSupported(provider PriceProvider, offer retailerOffer) bool {
	if provider == nil {
		return false
	}
	if capability, ok := provider.(interface{ Supports(ProviderListing) bool }); ok {
		return capability.Supports(ProviderListing{URL: offer.ListingURL, Seller: offer.Seller})
	}
	_, err := validateListing(offer.ListingURL)
	return err == nil
}
