package httpjson

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "invalid_name", "Name is required.")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "invalid_name" || body.Error.Message != "Name is required." {
		t.Fatalf("body = %+v", body)
	}
}

func decode(body string) error {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	return Decode(httptest.NewRecorder(), req, &dst)
}

func TestDecode(t *testing.T) {
	cases := map[string]bool{
		`{"name":"Laptop"}`:            true,
		`{"name":"Laptop","extra":1}`:  false,
		`{"name":"a"}{"name":"b"}`:     false,
		`not json`:                     false,
		``:                             false,
		`{"name":"Laptop"}` + "\n\n  ": true,
	}
	for body, ok := range cases {
		err := decode(body)
		if (err == nil) != ok {
			t.Errorf("Decode(%q) error = %v, want ok=%v", body, err, ok)
		}
	}
}
