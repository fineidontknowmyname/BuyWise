package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type measuredWriter struct {
	http.ResponseWriter
	status int
}

func (w *measuredWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *measuredWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *measuredWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func Observe(next http.Handler, logger *slog.Logger) http.Handler {
	var requests, failures atomic.Uint64
	var active atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/metrics" {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			_, _ = fmt.Fprintf(w, "# TYPE buywise_http_requests_total counter\nbuywise_http_requests_total %d\n# TYPE buywise_http_failures_total counter\nbuywise_http_failures_total %d\n# TYPE buywise_http_active gauge\nbuywise_http_active %d\n", requests.Load(), failures.Load(), active.Load())
			return
		}
		var id [12]byte
		_, _ = rand.Read(id[:])
		requestID := hex.EncodeToString(id[:])
		w.Header().Set("X-Request-ID", requestID)
		start := time.Now()
		active.Add(1)
		defer active.Add(-1)
		response := &measuredWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)
		requests.Add(1)
		status := response.status
		if status == 0 {
			status = 200
		}
		if status >= 500 {
			failures.Add(1)
		}
		if r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
			logger.Info("http request", "request_id", requestID, "method", r.Method, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}
	})
}
