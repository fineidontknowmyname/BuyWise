package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/modules/pricing"
	"selfemi/internal/platform/identity"
)

func TestIdentityIsolation(t *testing.T) {
	dsn := os.Getenv("BUYWISE_IDENTITY_TEST_DSN")
	if dsn == "" {
		t.Skip("requires a migrated disposable PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	app := NewHandler(Dependencies{
		DB: pool, Identity: identity.New(pool), GoalsRepo: goals.NewStore(pool),
		FinanceRepo: finance.NewStore(pool), PricingRepo: pricing.NewStore(pool),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: time.Now,
	})
	request := func(method, path, token, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response := httptest.NewRecorder()
		app.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("%s %s: got %d, want %d (%s)", method, path, response.Code, want, response.Body)
		}
		return response.Body.Bytes()
	}
	field := func(body []byte, name string) string {
		t.Helper()
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			t.Fatal(err)
		}
		value, ok := data[name].(string)
		if !ok || value == "" {
			t.Fatalf("missing %s", name)
		}
		return value
	}
	credentials := func(prefix string) string {
		return `{"email":"` + prefix + time.Now().UTC().Format("20060102150405.000000000") + `@example.test","password":"example-test-password"}`
	}
	a := credentials("one")
	b := credentials("two")
	request("GET", "/goals", "", "", 401)
	first := field(request("POST", "/auth/register", "", a, 200), "token")
	second := field(request("POST", "/auth/register", "", b, 200), "token")
	id := field(request("POST", "/goals", first, `{"name":"First account only"}`, 201), "id")
	request("GET", "/goals/"+id, first, "", 200)
	request("GET", "/goals/"+id, second, "", 404)
	request("PUT", "/goals/"+id, second, `{"name":"Changed"}`, 404)
	if data := request("GET", "/goals", second, "", 200); strings.Contains(string(data), id) {
		t.Fatal("the second account received the first account's goal")
	}
	if data := request("GET", "/categories", second, "", 200); !strings.Contains(string(data), "Salary") {
		t.Fatal("new account is missing default categories")
	}
	firstAccount := field(request("POST", "/accounts", first, `{"name":"Everyday","openingBalancePaise":0}`, 201), "id")
	request("POST", "/accounts", second, `{"name":"Everyday","openingBalancePaise":0}`, 201)
	request("GET", "/accounts/"+firstAccount, second, "", 404)
	secondUser := field(request("GET", "/auth/session", second, "", 200), "id")
	var legacyID string
	if err := pool.QueryRow(ctx, "INSERT INTO goals.goals (id, name, created_at) VALUES (uuidv7(), 'Unassigned legacy fixture', now()) RETURNING id::text").Scan(&legacyID); err != nil {
		t.Fatal(err)
	}
	request("GET", "/goals/"+legacyID, first, "", 404)
	request("GET", "/goals/"+legacyID, second, "", 404)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_tenant"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('buywise.user_id', $1, true)", secondUser); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO pricing.document_uploads
        (id, goal_id, filename, file_type, file_size, created_at)
        VALUES (uuidv7(), $1::uuid, 'fixture.csv', 'csv', 1, now())`, id)
	var foreignKey *pgconn.PgError
	if !errors.As(err, &foreignKey) || foreignKey.Code != "23503" || foreignKey.ConstraintName != "tenant_reference_9" {
		t.Fatalf("expected the same-owner foreign key to reject the link, got %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	request("GET", "/financial-profile", "", "", 401)
	profile := `{"revision":0,"variableIncomePercent":70,"bufferPercent":10,"bufferMinimumPaise":500000,"emergencyTargetPaise":1000000,"items":[
        {"name":"Salary","kind":"fixed_income","frequency":"monthly","amountPaise":6000000},
        {"name":"Freelance","kind":"variable_income","frequency":"monthly","amountPaise":1000000},
        {"name":"Bonus","kind":"irregular_income","frequency":"once","amountPaise":2000000},
        {"name":"Rent","kind":"fixed_expense","frequency":"monthly","amountPaise":2000000},
        {"name":"EMI","kind":"debt","frequency":"monthly","amountPaise":1000000},
        {"name":"Personal","kind":"discretionary","frequency":"monthly","amountPaise":300000}]}`
	request("PUT", "/financial-profile", first, profile, 200)
	request("PUT", "/financial-profile", first, profile, 409)
	assertJSON := func(body []byte, key, value string) {
		t.Helper()
		if !strings.Contains(string(body), "\""+key+"\":"+value) {
			t.Fatalf("expected %s=%s in %s", key, value, body)
		}
	}
	assertJSON(request("GET", "/financial-profile", first, "", 200), "safeSavingsPaise", "2900000")
	assertJSON(request("GET", "/financial-profile", second, "", 200), "revision", "0")
	donor := field(request("POST", "/accounts", first, `{"name":"Pay","openingBalancePaise":100000}`, 201), "id")
	other := field(request("POST", "/accounts", second, `{"name":"Other","openingBalancePaise":0}`, 201), "id")
	request("PUT", "/accounts/"+firstAccount+"/purpose", second, `{"purpose":"emergency_reserve"}`, 404)
	request("PUT", "/accounts/"+firstAccount+"/purpose", first, `{"purpose":"emergency_reserve"}`, 204)
	transfer := `{"fromAccountId":"` + donor + `","toAccountId":"` + firstAccount + `","amountPaise":25000,"occurredOn":"2020-01-01","idempotencyKey":"transfer-1"}`
	transferID := field(request("POST", "/transfers", first, transfer, 201), "id")
	if got := field(request("POST", "/transfers", first, transfer, 200), "id"); got != transferID {
		t.Fatal("retry created a second transfer")
	}
	request("POST", "/transfers", first, strings.Replace(transfer, "25000", "25001", 1), 409)
	request("POST", "/transfers", first, strings.ReplaceAll(strings.ReplaceAll(transfer, firstAccount, other), "transfer-1", "transfer-2"), 404)
	assertJSON(request("GET", "/accounts/"+donor, first, "", 200), "balancePaise", "75000")
	assertJSON(request("GET", "/accounts/"+firstAccount, first, "", 200), "balancePaise", "25000")
	assertJSON(request("GET", "/financial-profile", first, "", 200), "emergencySavingsPaise", "25000")
	if got := string(request("GET", "/transfers", second, "", 200)); strings.TrimSpace(got) != "[]" {
		t.Fatalf("second account could see transfers: %s", got)
	}
	request("POST", "/transfers/"+transferID+"/reverse", second, "", 404)
	reversed := field(request("POST", "/transfers/"+transferID+"/reverse", first, "", 200), "id")
	if got := field(request("POST", "/transfers/"+transferID+"/reverse", first, "", 200), "id"); got != reversed {
		t.Fatal("retry created a second reversal")
	}
	assertJSON(request("GET", "/accounts/"+donor, first, "", 200), "balancePaise", "100000")
	assertJSON(request("GET", "/accounts/"+firstAccount, first, "", 200), "balancePaise", "0")
	purchase := field(request("POST", "/goals", first, `{"name":"Laptop","targetPricePaise":8000000,"savedAmountPaise":2000000,"priority":"high"}`, 201), "id")
	planPath := "/goals/" + purchase + "/plan"
	fundingPath := "/goals/" + purchase + "/contributions"
	lifecyclePath := "/goals/" + purchase + "/lifecycle"
	request("GET", planPath, second, "", 404)
	assertJSON(request("GET", planPath, first, "", 200), "fundedPaise", "0")
	request("PUT", planPath, first, `{"revision":0,"mode":"intelligent","frequency":"monthly","approvedPaise":3000000}`, 400)
	terms := `{"revision":0,"mode":"intelligent","frequency":"monthly","approvedPaise":2000000}`
	request("PUT", planPath, first, terms, 200)
	request("PUT", planPath, first, terms, 409)
	request("PUT", "/goals/"+id, first, `{"name":"First account only","targetPricePaise":9000000,"priority":"medium"}`, 200)
	request("PUT", "/goals/"+id+"/plan", first, `{"revision":0,"mode":"simple","frequency":"monthly","approvedPaise":1500000}`, 200)
	assertJSON(request("GET", "/goals/"+id+"/plan", first, "", 200), "monthlyAllocatedPaise", "900000")
	assertJSON(request("POST", planPath+"/preview", first, `{"revision":1,"mode":"simple","frequency":"monthly","approvedPaise":10000}`, 200), "state", `"replan"`)
	contribution := `{"accountId":"` + donor + `","amountPaise":80000,"idempotencyKey":"goal-fund-1"}`
	request("POST", fundingPath, first, strings.Replace(contribution, donor, firstAccount, 1), 400)
	request("POST", fundingPath, first, contribution, 204)
	request("POST", fundingPath, first, contribution, 204)
	request("POST", fundingPath, first, strings.Replace(contribution, "80000", "80001", 1), 409)
	assertJSON(request("GET", "/accounts/"+donor, first, "", 200), "balancePaise", "100000")
	assertJSON(request("GET", planPath, first, "", 200), "fundedPaise", "80000")
	assertJSON(request("GET", "/goals/"+purchase, first, "", 200), "savedAmountPaise", "80000")
	request("PUT", lifecyclePath, first, `{"status":"paused"}`, 204)
	request("PUT", "/goals/"+purchase, first, `{"name":"Laptop revised","targetPricePaise":8000000,"priority":"high"}`, 200)
	assertJSON(request("GET", planPath, first, "", 200), "state", `"paused"`)
	request("POST", fundingPath, first, strings.Replace(contribution, "goal-fund-1", "goal-fund-2", 1), 409)
	request("PUT", lifecyclePath, first, `{"status":"active"}`, 204)
	request("PUT", lifecyclePath, first, `{"status":"cancelled"}`, 204)
	request("PUT", lifecyclePath, first, `{"status":"active"}`, 409)
	assertJSON(request("GET", planPath, first, "", 200), "fundedPaise", "80000")
	assertJSON(request("GET", "/goals/"+purchase, first, "", 200), "savedAmountPaise", "0")
	request("POST", "/auth/logout", first, "", 200)
	request("GET", "/goals", first, "", 401)
	retried := field(request("POST", "/auth/login", "", a, 200), "token")
	rotated := field(request("POST", "/auth/refresh", retried, "", 200), "token")
	request("POST", "/auth/refresh", retried, "", 401)
	request("GET", "/goals/"+id, retried, "", 401)
	request("GET", "/goals/"+id, rotated, "", 200)
}
