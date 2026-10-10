package finance

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newHTTP(t *testing.T) http.Handler {
	t.Helper()
	service := NewService(newMemoryRepo(), func() time.Time { return clockUTC })
	mux := http.NewServeMux()
	NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux)
	return mux
}

func call(t *testing.T, h http.Handler, method, path, body, key string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "{") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode %q: %v", rec.Body.String(), err)
		}
	}
	return rec.Code, out
}

func errorCode(out map[string]any) string {
	e, _ := out["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestHTTPFlow(t *testing.T) {
	h := newHTTP(t)
	code, account := call(t, h, "POST", "/accounts", `{"name":"Savings","openingBalancePaise":100000}`, "")
	if code != 201 || account["balancePaise"] != float64(100000) {
		t.Fatalf("account: %d %v", code, account)
	}
	accountID := account["id"].(string)

	_, cat := call(t, h, "POST", "/categories", `{"name":"Rent","kind":"expense","committed":true}`, "")
	categoryID := cat["id"].(string)

	body := `{"accountId":"` + accountID + `","categoryId":"` + categoryID + `","kind":"expense","amountPaise":25050,"occurredOn":"2026-10-01","note":"October"}`
	code, tx := call(t, h, "POST", "/transactions", body, "key-1")
	if code != 201 || tx["amountPaise"] != float64(25050) || tx["occurredOn"] != "2026-10-01" || tx["voided"] != false {
		t.Fatalf("create: %d %v", code, tx)
	}
	code, replay := call(t, h, "POST", "/transactions", body, "key-1")
	if code != 200 || replay["id"] != tx["id"] {
		t.Fatalf("replay: %d %v", code, replay)
	}

	_, got := call(t, h, "GET", "/accounts/"+accountID, "", "")
	if got["balancePaise"] != float64(100000-25050) {
		t.Fatalf("balance = %v", got["balancePaise"])
	}

	code, voided := call(t, h, "POST", "/transactions/"+tx["id"].(string)+"/void", `{"reason":"duplicate"}`, "")
	if code != 200 || voided["voided"] != true || voided["voidReason"] != "duplicate" {
		t.Fatalf("void: %d %v", code, voided)
	}
	code, again := call(t, h, "POST", "/transactions/"+tx["id"].(string)+"/void", "", "")
	if code != 409 || errorCode(again) != "already_voided" {
		t.Fatalf("second void: %d %v", code, again)
	}
	_, got = call(t, h, "GET", "/accounts/"+accountID, "", "")
	if got["balancePaise"] != float64(100000) {
		t.Fatalf("balance after void = %v", got["balancePaise"])
	}
}

func TestHTTPErrors(t *testing.T) {
	h := newHTTP(t)
	_, account := call(t, h, "POST", "/accounts", `{"name":"Main","openingBalancePaise":0}`, "")
	_, cat := call(t, h, "POST", "/categories", `{"name":"Food","kind":"expense","committed":false}`, "")
	valid := `{"accountId":"` + account["id"].(string) + `","categoryId":"` + cat["id"].(string) + `","kind":"expense","amountPaise":100}`

	cases := []struct {
		name, method, path, body, key string
		status                        int
		code                          string
	}{
		{"bad account json", "POST", "/accounts", `{"name":`, "", 400, "invalid_body"},
		{"unknown field", "POST", "/accounts", `{"name":"A","extra":1}`, "", 400, "invalid_body"},
		{"fractional paise", "POST", "/accounts", `{"name":"A","openingBalancePaise":10.5}`, "", 400, "invalid_body"},
		{"blank name", "POST", "/accounts", `{"name":" "}`, "", 400, "invalid_name"},
		{"duplicate account", "POST", "/accounts", `{"name":"main"}`, "", 409, "duplicate_name"},
		{"missing key", "POST", "/transactions", valid, "", 400, "invalid_idempotency_key"},
		{"zero amount", "POST", "/transactions", strings.Replace(valid, "100", "0", 1), "k", 400, "invalid_amount"},
		{"negative amount", "POST", "/transactions", strings.Replace(valid, "100", "-100", 1), "k", 400, "invalid_amount"},
		{"bad date", "POST", "/transactions", strings.Replace(valid, "}", `,"occurredOn":"2026-02-30"}`, 1), "k", 400, "invalid_date"},
		{"bad account id", "POST", "/transactions", strings.Replace(valid, account["id"].(string), "nope", 1), "k", 404, "account_not_found"},
		{"unknown account page", "GET", "/accounts/nope", "", "", 404, "account_not_found"},
		{"unknown transactions page", "GET", "/accounts/nope/transactions", "", "", 404, "account_not_found"},
		{"void unknown", "POST", "/transactions/nope/void", "", "", 404, "transaction_not_found"},
		{"committed income", "POST", "/categories", `{"name":"X","kind":"income","committed":true}`, "", 400, "invalid_committed"},
	}
	for _, c := range cases {
		status, out := call(t, h, c.method, c.path, c.body, c.key)
		if status != c.status || errorCode(out) != c.code {
			t.Errorf("%s: got %d %q, want %d %q", c.name, status, errorCode(out), c.status, c.code)
		}
	}
}

func TestHTTPKeyConflict(t *testing.T) {
	h := newHTTP(t)
	_, account := call(t, h, "POST", "/accounts", `{"name":"Main"}`, "")
	_, cat := call(t, h, "POST", "/categories", `{"name":"Food","kind":"expense"}`, "")
	base := `{"accountId":"` + account["id"].(string) + `","categoryId":"` + cat["id"].(string) + `","kind":"expense","amountPaise":`
	if code, _ := call(t, h, "POST", "/transactions", base+`100,"occurredOn":"2026-10-01"}`, "k"); code != 201 {
		t.Fatalf("first = %d", code)
	}
	code, out := call(t, h, "POST", "/transactions", base+`200,"occurredOn":"2026-10-01"}`, "k")
	if code != 409 || errorCode(out) != "idempotency_conflict" {
		t.Fatalf("conflict: %d %v", code, out)
	}
}
