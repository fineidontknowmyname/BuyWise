package identity

import (
	"bytes"
	"net/http"
)

type bufferedResponse struct {
	header  http.Header
	body    bytes.Buffer
	status  int
	started bool
}

func newResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), status: http.StatusOK}
}

func (r *bufferedResponse) Header() http.Header { return r.header }

func (r *bufferedResponse) WriteHeader(status int) {
	if !r.started {
		r.status, r.started = status, true
	}
}

func (r *bufferedResponse) Write(data []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	return r.body.Write(data)
}

func (r *bufferedResponse) writeTo(w http.ResponseWriter) {
	for key, values := range r.header {
		w.Header()[key] = values
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body.Bytes())
}
