package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func HTTPMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		if AppMetrics != nil {
			AppMetrics.ActiveConnections.Inc()
			defer AppMetrics.ActiveConnections.Dec()
		}
		
		next.ServeHTTP(wrapped, r)
		
		if AppMetrics != nil {
			duration := time.Since(start)
			route := getRoutePattern(r)
			method := r.Method
			status := strconv.Itoa(wrapped.statusCode)
			
			HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
			HTTPRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
			HTTPRequestSize.WithLabelValues(method, route).Observe(float64(r.ContentLength))
			HTTPResponseSize.WithLabelValues(method, route).Observe(float64(wrapped.size))
			
			if wrapped.statusCode >= 400 {
				HTTPErrors.WithLabelValues(method, route, status).Inc()
			}
		}
	})
}

func getRoutePattern(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return r.URL.Path
	}
	
	template, err := route.GetPathTemplate()
	if err != nil {
		return r.URL.Path
	}
	
	return template
}

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "echofs_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "route", "status"})
	
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "echofs_http_request_duration_seconds",
		Help: "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
	
	HTTPRequestSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "echofs_http_request_size_bytes",
		Help: "Size of HTTP requests",
		Buckets: prometheus.ExponentialBuckets(1, 2, 20),
	}, []string{"method", "route"})
	
	HTTPResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "echofs_http_response_size_bytes",
		Help: "Size of HTTP responses",
		Buckets: prometheus.ExponentialBuckets(1, 2, 20),
	}, []string{"method", "route"})
	
	HTTPErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "echofs_http_errors_total",
		Help: "Total number of HTTP errors",
	}, []string{"method", "route", "status"})
)