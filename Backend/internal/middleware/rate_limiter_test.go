package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowsRequestsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(10, time.Second, 10)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow 10 requests (burst capacity)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksExcessRequests(t *testing.T) {
	rl := NewRateLimiter(5, time.Second, 5)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the bucket
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimiter_DifferentIPsHaveSeparateBuckets(t *testing.T) {
	rl := NewRateLimiter(2, time.Second, 2)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// IP 2 should still work
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got %d", rec.Code)
	}
}

func TestRateLimiter_TokensRefillOverTime(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond, 2)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust bucket
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "3.3.3.3:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should work again
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "3.3.3.3:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after token refill, got %d", rec.Code)
	}
}

func TestRateLimiter_XForwardedForExtractsClientIP(t *testing.T) {
	rl := NewRateLimiter(1, time.Second, 1)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request from proxy with X-Forwarded-For
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "proxy:8080"
	req1.Header.Set("X-Forwarded-For", "real-client, proxy1, proxy2")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("first request should succeed, got %d", rec1.Code)
	}

	// Second request from same real client should be limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "proxy:8080"
	req2.Header.Set("X-Forwarded-For", "real-client, proxy1, proxy2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("second request from same client should be limited, got %d", rec2.Code)
	}
}

func TestRateLimiter_RetryAfterHeaderSet(t *testing.T) {
	rl := NewRateLimiter(1, time.Second, 1)

	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "4.4.4.4:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Rate limited request should have Retry-After header
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "4.4.4.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req2)

	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on rate limited response")
	}
}
