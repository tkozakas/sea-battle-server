package transport

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHijackFlusher struct {
	http.ResponseWriter
	hijacked bool
	flushed  bool
}

func (f *fakeHijackFlusher) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true
	return nil, nil, nil
}

func (f *fakeHijackFlusher) Flush() {
	f.flushed = true
}

func newResponseWriter(inner http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: inner, status: http.StatusOK}
}

func TestResponseWriterWriteHeaderRecordsStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusTeapot)

	if rw.status != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, rw.status)
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("expected underlying recorder code %d, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestResponseWriterUnwrapReturnsUnderlying(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	if rw.Unwrap() != rec {
		t.Error("Unwrap did not return the underlying ResponseWriter")
	}
}

func TestResponseWriterHijackDelegatesToUnderlying(t *testing.T) {
	fake := &fakeHijackFlusher{ResponseWriter: httptest.NewRecorder()}
	rw := newResponseWriter(fake)

	_, _, err := rw.Hijack()
	if err != nil {
		t.Fatalf("unexpected error from Hijack: %v", err)
	}
	if !fake.hijacked {
		t.Error("expected Hijack to be delegated to underlying writer")
	}
}

func TestResponseWriterHijackReturnsErrorWhenNotSupported(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	_, _, err := rw.Hijack()
	if err == nil {
		t.Error("expected error when underlying writer does not support Hijack")
	}
}

func TestResponseWriterFlushDelegatesToUnderlying(t *testing.T) {
	fake := &fakeHijackFlusher{ResponseWriter: httptest.NewRecorder()}
	rw := newResponseWriter(fake)

	rw.Flush()

	if !fake.flushed {
		t.Error("expected Flush to be delegated to underlying writer")
	}
}

func TestResponseWriterFlushNoopWhenNotSupported(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.Flush()
}

func TestCORSMiddlewareWildcardSetsAllowOriginStar(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", got)
	}
}

func TestCORSMiddlewareWildcardSetsMethodsAndHeaders(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}
}

func TestCORSMiddlewareAllowsMatchingOrigin(t *testing.T) {
	allowed := "https://app.example.com"
	handler := CORSMiddleware([]string{allowed})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", allowed)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != allowed {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", allowed, got)
	}
}

func TestCORSMiddlewareBlocksNonMatchingOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for disallowed origin, got %q", got)
	}
}

func TestCORSMiddlewareOptionsReturnsNoContent(t *testing.T) {
	handler := CORSMiddleware([]string{"*"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d for OPTIONS, got %d", http.StatusNoContent, rec.Code)
	}
}
