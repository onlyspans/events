package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := Logging(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"method":"GET"`) {
		t.Error("expected log to contain method")
	}
	if !strings.Contains(logOutput, `"path":"/test"`) {
		t.Error("expected log to contain path")
	}
	if !strings.Contains(logOutput, `"status":200`) {
		t.Error("expected log to contain status")
	}
}

func TestLogging_CapturesNonOKStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	middleware := Logging(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"status":404`) {
		t.Error("expected log to contain status 404")
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recovery(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Error("expected log to contain panic message")
	}
	if !strings.Contains(logOutput, "test panic") {
		t.Error("expected log to contain panic value")
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := Recovery(logger)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if buf.Len() > 0 {
		t.Error("expected no log output when no panic")
	}
}

func TestChain(t *testing.T) {
	var order []string

	first := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "first-before")
			next.ServeHTTP(w, r)
			order = append(order, "first-after")
		})
	}

	second := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "second-before")
			next.ServeHTTP(w, r)
			order = append(order, "second-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain(first, second)
	wrapped := chain(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// First middleware wraps second, which wraps handler
	expected := []string{"first-before", "second-before", "handler", "second-after", "first-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("at position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestChain_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := Chain()
	wrapped := chain(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
