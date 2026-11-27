package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const serpAPIOrigin = "https://serpapi.com"

var amazonASIN = regexp.MustCompile(`(?i)/(?:dp|gp/product)/([a-z0-9]{10})(?:/|$)`)
var rupeePrice = regexp.MustCompile(`^(?:₹|INR)\s*([0-9]+(?:,[0-9]+)*)(?:\.([0-9]{1,2}))?$`)

var errProviderUnsupported = errors.New("listing unsupported by price provider")
var errProviderNoPrice = errors.New("provider returned no matching price")
var errProviderQuota = errors.New("price provider budget exhausted")
var errProviderCredentials = errors.New("price provider credentials rejected")

type SerpAPI struct {
	pool      *pgxpool.Pool
	client    *http.Client
	origin    string
	key       string
	previewMu sync.Mutex
	previews  map[string]cachedListingPreview
}

func NewSerpAPI(pool *pgxpool.Pool, key string) *SerpAPI {
	return &SerpAPI{pool: pool, origin: serpAPIOrigin, key: key, client: &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func amazonListingASIN(raw string) string {
	link, err := validateListing(raw)
	if err != nil || link.Retailer != "Amazon" {
		return ""
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		return ""
	}
	for key := range u.Query() {
		switch strings.ToLower(key) {
		case "smid", "seller", "sellerid", "m", "merchantid", "offerlistingid", "condition":
			return ""
		}
	}
	matches := amazonASIN.FindStringSubmatch(u.Path)
	if matches == nil {
		return ""
	}
	return strings.ToUpper(matches[1])
}

func (s *SerpAPI) Supports(listing ProviderListing) bool {
	return strings.TrimSpace(listing.Seller) == "" && amazonListingASIN(listing.URL) != ""
}

func (s *SerpAPI) get(ctx context.Context, path string, params url.Values, result any) error {
	params.Set("api_key", s.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.origin+path+"?"+params.Encode(), nil)
	if err != nil {
		return ErrProviderUnavailable
	}
	res, err := s.client.Do(req)
	if err != nil {
		return ErrProviderUnavailable
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return errProviderCredentials
	case http.StatusTooManyRequests:
		return ErrProviderRateLimited
	case http.StatusOK:
		if json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(result) == nil {
			return nil
		}
	}
	return ErrProviderUnavailable
}

type serpProduct struct {
	Metadata struct {
		Status  string `json:"status"`
		Created string `json:"created_at"`
	} `json:"search_metadata"`
	Product struct {
		Title      string `json:"title"`
		Brand      string `json:"brand"`
		Thumbnail  string `json:"thumbnail"`
		Categories []struct {
			Name string `json:"name"`
		} `json:"categories"`
		Variants []struct {
			Title string `json:"title"`
			Items []struct {
				Name     string `json:"name"`
				Selected bool   `json:"selected"`
			} `json:"items"`
		} `json:"variants"`
		ASIN  string `json:"asin"`
		Link  string `json:"link_clean"`
		Stock string `json:"stock"`
	} `json:"product_results"`
	Purchase struct {
		BuyNew struct {
			Price string `json:"price"`
			Stock string `json:"stock"`
		} `json:"buy_new"`
	} `json:"purchase_options"`
}

func (s *SerpAPI) fetchPrice(ctx context.Context, asin string) (ObservedOfferPrice, error) {
	var result serpProduct
	err := s.get(ctx, "/search.json", url.Values{
		"engine": {"amazon_product"}, "amazon_domain": {"amazon.in"}, "asin": {asin},
	}, &result)
	if err != nil {
		return ObservedOfferPrice{}, err
	}
	if result.Metadata.Status != "Success" {
		return ObservedOfferPrice{}, ErrProviderUnavailable
	}
	return result.observation(asin, time.Now().UTC())
}

func (p serpProduct) observation(asin string, now time.Time) (ObservedOfferPrice, error) {
	if !strings.EqualFold(p.Product.ASIN, asin) || (p.Product.Link != "" && amazonListingASIN(p.Product.Link) != asin) {
		return ObservedOfferPrice{}, errProviderNoPrice
	}
	item := p.Purchase.BuyNew
	paise := parseSerpRupees(item.Price)
	observed, err := time.Parse("2006-01-02 15:04:05 MST", p.Metadata.Created)
	if err != nil {
		observed, err = time.Parse(time.RFC3339, p.Metadata.Created)
	}
	if paise == 0 || err != nil || observed.Before(now.Add(-24*time.Hour)) || observed.After(now) {
		return ObservedOfferPrice{}, errProviderNoPrice
	}
	stock := strings.ToLower(strings.TrimSpace(item.Stock))
	if stock == "" {
		stock = strings.ToLower(strings.TrimSpace(p.Product.Stock))
	}
	availability := "unknown"
	switch {
	case strings.Contains(stock, "out of stock"), strings.Contains(stock, "unavailable"):
		availability = "out_of_stock"
	case strings.Contains(stock, "pre-order"), strings.Contains(stock, "preorder"):
		availability = "pre_order"
	case stock == "in stock", strings.HasPrefix(stock, "only ") && strings.Contains(stock, "left in stock"):
		availability = "in_stock"
	}
	return ObservedOfferPrice{ObservedOn: marketDay(observed), ObservedAt: &observed, SourceLabel: serpSource,
		Availability: availability, ItemPaise: paise}, nil
}

func parseSerpRupees(price string) int64 {
	match := rupeePrice.FindStringSubmatch(strings.TrimSpace(price))
	if match == nil {
		return 0
	}
	rupees, err := strconv.ParseInt(strings.ReplaceAll(match[1], ",", ""), 10, 64)
	if err != nil || rupees > maxOfferPaise/100 {
		return 0
	}
	fraction, _ := strconv.ParseInt((match[2] + "00")[:2], 10, 64)
	total := 100*rupees + fraction
	if total > maxOfferPaise {
		return 0
	}
	return total
}
