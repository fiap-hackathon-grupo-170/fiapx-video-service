package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitTracer_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tp, err := InitTracer(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil tracer provider")
	}
	tp.Shutdown(context.Background())
}
