package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func get(t *testing.T, handler http.Handler, path string) (int, healthResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	return rec.Code, body
}

func TestLivenessIsUpEvenWhenDatabaseIsDown(t *testing.T) {
	code, body := get(t, NewRouter(fakePinger{err: errors.New("down")}), "/healthz")
	if code != http.StatusOK || body.Status != "up" {
		t.Fatalf("got %d %+v", code, body)
	}
}

func TestReadinessWithDatabaseUp(t *testing.T) {
	code, body := get(t, NewRouter(fakePinger{}), "/readyz")
	if code != http.StatusOK || body.Status != "up" || body.Database != "up" {
		t.Fatalf("got %d %+v", code, body)
	}
}

func TestReadinessWithDatabaseDown(t *testing.T) {
	code, body := get(t, NewRouter(fakePinger{err: errors.New("down")}), "/readyz")
	if code != http.StatusServiceUnavailable || body.Status != "degraded" || body.Database != "down" {
		t.Fatalf("got %d %+v", code, body)
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(fakePinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestHealthRejectsWrongMethod(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(fakePinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d", rec.Code)
	}
}
