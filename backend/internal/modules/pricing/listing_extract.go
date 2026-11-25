package pricing

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"selfemi/internal/platform/httpjson"
)

type listingPreview struct {
	URL                   string     `json:"url"`
	Retailer              string     `json:"retailer"`
	Name                  string     `json:"name,omitempty"`
	Brand                 string     `json:"brand,omitempty"`
	Variant               string     `json:"variant,omitempty"`
	Category              string     `json:"category,omitempty"`
	ImageURL              string     `json:"imageUrl,omitempty"`
	CurrentPricePaise     int64      `json:"currentPricePaise,omitempty"`
	ObservedAt            *time.Time `json:"observedAt,omitempty"`
	Source                string     `json:"source,omitempty"`
	ManualDetailsRequired bool       `json:"manualDetailsRequired"`
	Warnings              []string   `json:"warnings"`
}

type listingRetriever interface {
	LookupListing(context.Context, string) (listingPreview, error)
}

func (h offerHandler) extractListing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		h.fail(w, errUnsupportedListing)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	link, err := resolveRetailerLink(ctx, input.URL, newShortLinkClient())
	if err != nil {
		if errors.Is(err, errUnsupportedListing) {
			h.fail(w, err)
			return
		}
		httpjson.WriteError(w, 422, "short_link_unresolved", "Could not resolve the Amazon short link. Open it in your browser and paste the full amazon.in product URL.")
		return
	}
	preview := listingPreview{URL: link.URL, Retailer: link.Retailer, ManualDetailsRequired: true, Warnings: []string{}}
	provider, available := h.provider.(listingRetriever)
	switch {
	case !available:
		preview.Warnings = append(preview.Warnings, "Product retrieval is unavailable. Enter details manually; check the server's SerpAPI configuration.")
	case link.Retailer != "Amazon" || amazonListingASIN(link.URL) == "":
		preview.Warnings = append(preview.Warnings, "Automatic retrieval supports direct Amazon India product links without seller overrides. Enter this listing's details manually.")
	default:
		result, lookupErr := provider.LookupListing(ctx, link.URL)
		if lookupErr == nil {
			preview = result
		} else {
			message := "The product provider could not return matching details. Enter them manually or try again later."
			switch {
			case errors.Is(lookupErr, errProviderQuota):
				message = "The product lookup allowance is exhausted. Enter details manually."
			case errors.Is(lookupErr, errProviderCredentials):
				message = "The product provider rejected its API key. Check the server configuration or enter details manually."
			case errors.Is(lookupErr, ErrProviderRateLimited):
				message = "The product provider is busy. Try again later or enter details manually."
			}
			preview.Warnings = append(preview.Warnings, message)
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpjson.Write(w, 200, preview)
}

func safeProductImage(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2000 || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return ""
	}
	switch strings.ToLower(u.Hostname()) {
	case "m.media-amazon.com", "images-eu.ssl-images-amazon.com", "images-na.ssl-images-amazon.com":
		return u.String()
	}
	return ""
}

func shortLinkURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || len(raw) > 2000 || u.Scheme != "https" || u.User != nil || u.Port() != "" || u.Opaque != "" || u.Path == "" || u.Path == "/" {
		return nil, errUnsupportedListing
	}
	switch strings.ToLower(u.Hostname()) {
	case "amzn.in", "amzn.to", "amazon.in", "www.amazon.in":
		u.Host = strings.ToLower(u.Host)
		return u, nil
	}
	return nil, errUnsupportedListing
}

func resolveRetailerLink(ctx context.Context, raw string, client *http.Client) (listingLink, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return listingLink{}, errUnsupportedListing
	}
	if !strings.EqualFold(u.Hostname(), "amzn.in") && !strings.EqualFold(u.Hostname(), "amzn.to") {
		return validateListing(raw)
	}
	u, err = shortLinkURL(raw)
	if err != nil {
		return listingLink{}, err
	}
	for hops := 0; hops < 5; hops++ {
		if strings.EqualFold(u.Hostname(), "amazon.in") || strings.EqualFold(u.Hostname(), "www.amazon.in") {
			return validateListing(u.String())
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return listingLink{}, errUnsupportedListing
		}
		res, err := client.Do(req)
		if err != nil {
			return listingLink{}, ErrProviderUnavailable
		}
		res.Body.Close()
		if res.StatusCode != 301 && res.StatusCode != 302 && res.StatusCode != 303 && res.StatusCode != 307 && res.StatusCode != 308 {
			return listingLink{}, ErrProviderUnavailable
		}
		next, err := res.Location()
		if err != nil {
			return listingLink{}, ErrProviderUnavailable
		}
		u, err = shortLinkURL(next.String())
		if err != nil {
			return listingLink{}, err
		}
	}
	return listingLink{}, ErrProviderUnavailable
}

func publicListingAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	if address.Is6() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, prefix := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96", "64:ff9b:1::/48"} {
		if netip.MustParsePrefix(prefix).Contains(address) {
			return false
		}
	}
	return true
}

func newShortLinkClient() *http.Client {
	transport := &http.Transport{
		MaxResponseHeaderBytes: 16 << 10,
		DisableKeepAlives:      true,
		TLSHandshakeTimeout:    3 * time.Second,
		DialContext: func(ctx context.Context, network, endpoint string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(endpoint)
			if err != nil || port != "443" || (host != "amzn.in" && host != "amzn.to") {
				return nil, errUnsupportedListing
			}
			addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil || len(addresses) == 0 {
				return nil, ErrProviderUnavailable
			}
			for _, address := range addresses {
				if !publicListingAddress(address) {
					return nil, errUnsupportedListing
				}
			}
			dialer := net.Dialer{Timeout: 3 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
		},
	}
	return &http.Client{Transport: transport, Timeout: 6 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}
