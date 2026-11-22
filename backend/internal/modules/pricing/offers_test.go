package pricing

import (
	"strings"
	"testing"
	"time"
)

func TestOfferListingAndPriceValidation(t *testing.T) {
	listing, err := validateListing("https://amazon.in/dp/EXAMPLE?tag=test&variant=16#details")
	if err != nil || listing.Retailer != "Amazon" || listing.URL != "https://www.amazon.in/dp/EXAMPLE?variant=16" {
		t.Fatalf("unexpected listing %+v, err=%v", listing, err)
	}
	for _, value := range []string{"", "https://127.0.0.1/", "https://169.254.169.254/", "https://example.com/store", "http://amazon.in/dp/EXAMPLE"} {
		if _, err := validateListing(value); err == nil {
			t.Fatalf("accepted unsupported listing: %s", value)
		}
	}
	now := time.Date(2026, 1, 9, 8, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	observation := ObservedOfferPrice{ObservedOn: "2026-01-09", ObservedAt: &future, ItemPaise: 100000}
	if observation.validate(now) == nil {
		t.Fatal("future observations must be rejected")
	}
	if priceFreshness("", now) != "missing" || priceFreshness("2026-01-05", now) != "stale" ||
		priceFreshness("2026-01-09", now) != "recent" {
		t.Fatal("unexpected freshness")
	}
}

func TestOfferCSVUsesExactPaise(t *testing.T) {
	input := strings.Join([]string{
		"date,price,source",
		`2020-01-02,"₹1,20,000.05","Receipt, checked"`,
		"02/01/2020,120001.1,Shop",
	}, "\n")
	records, err := parseOfferCSV(input)
	if err != nil || len(records) != 2 || records[0].ItemPaise != 12000005 ||
		records[1].ItemPaise != 12000110 || records[0].SourceLabel != "Receipt, checked" ||
		records[1].ObservedOn != "2020-01-02" || records[0].ObservedAt != nil {
		t.Fatalf("unexpected parsed observations %+v, err=%v", records, err)
	}
	if _, err := parseOfferCSV("date,price\n2020-01-02,12.3extra"); err == nil {
		t.Fatal("accepted malformed price")
	}
}
