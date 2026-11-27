package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"selfemi/internal/modules/goals"
)

type memoryRepo struct {
	mu    sync.Mutex
	items []goals.Goal
}

func (m *memoryRepo) Create(_ context.Context, goal goals.Goal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, goal)
	return nil
}

func (m *memoryRepo) Get(_ context.Context, id uuid.UUID) (goals.Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, g := range m.items {
		if g.ID == id {
			return g, nil
		}
	}
	return goals.Goal{}, goals.ErrNotFound
}

func (m *memoryRepo) List(context.Context) ([]goals.Goal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]goals.Goal{}, m.items...), nil
}

func (m *memoryRepo) Update(_ context.Context, goal goals.Goal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, g := range m.items {
		if g.ID == goal.ID {
			m.items[i] = goal
			return nil
		}
	}
	return goals.ErrNotFound
}

type upDB struct{}

func (upDB) Ping(context.Context) error { return nil }

var fixedNow = time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)

func newTestHandler() http.Handler {
	return newDomainHandler(Dependencies{
		DB:        upDB{},
		GoalsRepo: &memoryRepo{},
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:       func() time.Time { return fixedNow },
	})
}

func do(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	var out map[string]any
	if rec.Body.Len() > 0 && strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "{") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return rec.Code, out
}

func TestCreateGoalThenDecisionWithoutEvidence(t *testing.T) {
	h := newTestHandler()

	code, goal := do(t, h, http.MethodPost, "/goals", `{"name":"  Laptop  "}`)
	if code != http.StatusCreated {
		t.Fatalf("create code = %d, body %v", code, goal)
	}
	if goal["name"] != "Laptop" {
		t.Fatalf("name = %v", goal["name"])
	}
	id, _ := goal["id"].(string)
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("id %q is not a UUID: %v", id, err)
	}
	if v := parsed[6] >> 4; v != 7 {
		t.Fatalf("id version = %d, want 7", v)
	}

	code, d := do(t, h, http.MethodGet, "/goals/"+id+"/decision", "")
	if code != http.StatusOK {
		t.Fatalf("decision code = %d, body %v", code, d)
	}
	if d["state"] != "REASSESS" || d["rulesVersion"] != "purchase-v2" || d["trigger"] != "live_preview" {
		t.Fatalf("decision = %v", d)
	}
	reasons, _ := d["reasonCodes"].([]any)
	if len(reasons) != 2 || reasons[0] != "profile_missing" || reasons[1] != "product_missing" {
		t.Fatalf("unexpected reasons: %v", reasons)
	}
	if d["decidedAt"] != "2026-10-09T10:00:00Z" {
		t.Errorf("decidedAt = %v", d["decidedAt"])
	}
}

func TestCreateGoalRejectsBadInput(t *testing.T) {
	h := newTestHandler()
	cases := map[string]string{
		`{"name":""}`:    "invalid_name",
		`{"name":"   "}`: "invalid_name",
		`{"name":"` + strings.Repeat("a", 201) + `"}`: "invalid_name",
		`{"name":"x","extra":true}`:                   "invalid_body",
		`not json`:                                    "invalid_body",
	}
	for body, wantCode := range cases {
		code, out := do(t, h, http.MethodPost, "/goals", body)
		errBody, _ := out["error"].(map[string]any)
		if code != http.StatusBadRequest || errBody["code"] != wantCode {
			t.Errorf("body %.30q: got %d %v, want 400 %s", body, code, out, wantCode)
		}
	}
}

func TestNameLimitCountsCharactersNotBytes(t *testing.T) {
	h := newTestHandler()
	code, _ := do(t, h, http.MethodPost, "/goals", `{"name":"`+strings.Repeat("₹", 200)+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("200 multi-byte characters should be accepted, got %d", code)
	}
}

func TestUnknownGoal(t *testing.T) {
	h := newTestHandler()
	for _, path := range []string{
		"/goals/" + uuid.NewV7().String(),
		"/goals/" + uuid.NewV7().String() + "/decision",
		"/goals/not-a-uuid",
		"/goals/not-a-uuid/decision",
	} {
		code, out := do(t, h, http.MethodGet, path, "")
		errBody, _ := out["error"].(map[string]any)
		if code != http.StatusNotFound || errBody["code"] != "goal_not_found" {
			t.Errorf("%s: got %d %v", path, code, out)
		}
	}
}

func TestListGoals(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/goals", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %q", rec.Code, rec.Body.String())
	}
	do(t, h, http.MethodPost, "/goals", `{"name":"Laptop"}`)
	do(t, h, http.MethodPost, "/goals", `{"name":"Fridge"}`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/goals", nil))
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("list = %s, err %v", rec.Body.String(), err)
	}
}
