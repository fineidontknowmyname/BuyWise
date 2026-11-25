package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/platform/identity"
)

func TestSerpAPIPriceIdentity(t *testing.T) {
	now := time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC)
	const asin = "B072MQ5BRX"
	var result serpProduct
	result.Metadata.Created = "2026-11-20 18:59:00 UTC"
	result.Product.ASIN = asin
	result.Product.Link = "https://www.amazon.in/dp/" + asin
	result.Purchase.BuyNew.Price = "₹1,23,456.75"
	result.Purchase.BuyNew.Stock = "Only 4 left in stock"
	price, err := result.observation(asin, now)
	if err != nil || price.ItemPaise != 12345675 || price.Availability != "in_stock" || price.ObservedOn != "2026-11-21" ||
		price.TaxPaise != nil || price.DeliveryPaise != nil || price.DiscountPaise != nil {
		t.Fatalf("unexpected observation: %+v, %v", price, err)
	}
	for _, bad := range []string{"$123", "₹120–₹150", "₹0.00", "₹1.234", "₹9999999999999999"} {
		result.Purchase.BuyNew.Price = bad
		if _, err := result.observation(asin, now); !errors.Is(err, errProviderNoPrice) {
			t.Errorf("accepted invalid price %q: %v", bad, err)
		}
	}
	result.Purchase.BuyNew.Price = "INR 99.7"
	for _, bad := range []string{"https://www.amazon.in/dp/B073CX2PDW", "https://www.amazon.com/dp/" + asin} {
		result.Product.Link = bad
		if _, err := result.observation(asin, now); !errors.Is(err, errProviderNoPrice) {
			t.Errorf("accepted mismatched listing %q: %v", bad, err)
		}
	}
	provider := &SerpAPI{}
	if !provider.Supports(ProviderListing{URL: "https://amazon.in/gp/product/b072mq5brx?tag=affiliate"}) ||
		provider.Supports(ProviderListing{URL: "https://amazon.in/dp/" + asin + "?smid=SellerID"}) ||
		provider.Supports(ProviderListing{URL: "https://amazon.in/dp/" + asin, Seller: "Specific seller"}) ||
		provider.Supports(ProviderListing{URL: "https://flipkart.com/dp/" + asin}) {
		t.Fatal("incorrect provider eligibility")
	}
}

func TestSerpAPISharedBudget(t *testing.T) {
	dsn := os.Getenv("BUYWISE_SERPAPI_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a freshly migrated disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var productCalls atomic.Int32
	var used atomic.Int32
	used.Store(219)
	var renewal atomic.Int32
	renewal.Store(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "fixture" {
			http.Error(w, "invalid fixture request", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/account.json":
			usage := used.Load()
			fmt.Fprintf(w, `{"plan_renewal_date":"2040-01-%02d","searches_per_month":250,"plan_searches_left":%d,"this_month_usage":%d}`, renewal.Load(), 250-usage, usage)
		case "/search.json":
			productCalls.Add(1)
			query := r.URL.Query()
			if query.Get("engine") != "amazon_product" || query.Get("amazon_domain") != "amazon.in" || query.Get("no_cache") != "" {
				t.Error("incorrect Amazon India query")
				w.WriteHeader(400)
				return
			}
			asin := query.Get("asin")
			stamp := time.Now().UTC().Add(-time.Minute).Format("2006-01-02 15:04:05 MST")
			fmt.Fprintf(w, `{"search_metadata":{"status":"Success","created_at":%q},"product_results":{"asin":%q},"purchase_options":{"buy_new":{"price":"₹1,299.90","stock":"In Stock"}}}`, stamp, asin)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	newProvider := func() *SerpAPI {
		provider := NewSerpAPI(pool, "fixture")
		provider.origin = server.URL
		return provider
	}
	observe := func(provider *SerpAPI, asin string, scheduled bool) (ObservedOfferPrice, error) {
		return provider.Observe(ctx, ProviderListing{URL: "https://www.amazon.in/dp/" + asin, Scheduled: scheduled})
	}
	const first = "B000000001"
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			price, err := observe(newProvider(), first, true)
			if err == nil && price.ItemPaise != 129990 {
				err = fmt.Errorf("unexpected price: %d", price.ItemPaise)
			}
			errs <- err
		}()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if got := productCalls.Load(); got != 1 {
		t.Fatalf("expected one shared product lookup, got %d", got)
	}
	const second = "B000000002"
	if _, err := observe(newProvider(), second, true); !errors.Is(err, errProviderQuota) {
		t.Fatalf("scheduled refresh used the manual reserve: %v", err)
	}
	used.Store(249)
	if _, err := observe(newProvider(), second, false); err != nil {
		t.Fatal(err)
	}
	const third = "B000000003"
	if _, err := observe(newProvider(), third, false); !errors.Is(err, errProviderQuota) {
		t.Fatalf("expected the hard cap to block lookups: %v", err)
	}
	renewal.Store(2)
	used.Store(0)
	if _, err := observe(newProvider(), third, true); err != nil {
		t.Fatalf("new billing cycle did not reset the budget: %v", err)
	}
	if got := productCalls.Load(); got != 3 {
		t.Fatalf("expected three product lookups, got %d", got)
	}
	testSerpQueuedRefresh(t, ctx, pool, newProvider(), third)
}

func testSerpQueuedRefresh(t *testing.T, ctx context.Context, pool *pgxpool.Pool, provider *SerpAPI, asin string) {
	t.Helper()
	var owner, productID, offerID string
	email := fmt.Sprintf("serpapi-%d@example.test", time.Now().UnixNano())
	mux := http.NewServeMux()
	identity.New(pool).Register(mux)
	credentials := fmt.Sprintf(`{"email":%q,"password":"example-test-password"}`, email)
	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest("POST", "/auth/register", strings.NewReader(credentials)))
	var session struct {
		Token string `json:"token"`
	}
	if login.Code != 200 || json.Unmarshal(login.Body.Bytes(), &session) != nil || session.Token == "" {
		t.Fatalf("could not register fixture account: HTTP %d", login.Code)
	}
	request := httptest.NewRequest("GET", "/auth/session", nil)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	info := httptest.NewRecorder()
	mux.ServeHTTP(info, request)
	var user map[string]string
	if info.Code != 200 || json.Unmarshal(info.Body.Bytes(), &user) != nil || user["id"] == "" {
		t.Fatalf("could not read fixture session: HTTP %d", info.Code)
	}
	owner = user["id"]
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if err := scopeRefreshTenant(ctx, tx, owner); err != nil {
		t.Fatal(err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO pricing.products (id, name, created_at, updated_at)
        VALUES (uuidv7(), 'Fixture', now(), now()) RETURNING id::text`).Scan(&productID)
	if err != nil {
		t.Fatal(err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO pricing.retailer_offers (product_id, retailer, seller, listing_url)
        VALUES ($1, 'Amazon', '', $2) RETURNING id::text`, productID, "https://www.amazon.in/dp/"+asin).Scan(&offerID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO pricing.target_thresholds (product_id, good_paise) VALUES ($1, 130000)", productID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO pricing.offer_refresh_jobs (offer_id) VALUES ($1)", offerID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	worker := NewRefreshWorker(pool, provider, func() time.Time { return now })
	if err := worker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var next time.Time
	err = pool.QueryRow(ctx, `SELECT state, next_refresh_at FROM pricing.offer_refresh_jobs WHERE offer_id = $1`, offerID).Scan(&state, &next)
	if err != nil || state != "succeeded" || !next.Equal(now.Add(24*time.Hour).Truncate(time.Microsecond)) {
		t.Fatalf("expected daily refresh after successful observation, got %s at %v: %v", state, next, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE pricing.offer_refresh_jobs SET next_refresh_at = now() WHERE offer_id = $1`, offerID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := worker.scheduleDue(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var jobs int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM pricing.offer_refresh_jobs WHERE offer_id = $1 AND scheduled AND state = 'queued'`, offerID).Scan(&jobs)
	if err != nil || jobs != 1 {
		t.Fatalf("expected exactly one scheduled refresh, got %d: %v", jobs, err)
	}
	if err := worker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	var records int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM pricing.offer_observations WHERE offer_id = $1 AND source = 'provider'`, offerID).Scan(&records)
	if err != nil || records != 1 {
		t.Fatalf("expected one idempotent observation, got %d: %v", records, err)
	}
}
