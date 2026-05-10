// Gateway service — authentication, rate limiting, request routing.
// This is the public-facing entry point for all client requests.
// It validates JWT tokens and forwards requests to the appropriate backend service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"echofs/internal/middleware"
	"echofs/pkg/auth"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type Gateway struct {
	router       *mux.Router
	logger       *log.Logger
	jwtManager   *auth.JWTManager
	authMW       *auth.AuthMiddleware
	rateLimiter  *middleware.RateLimiter
	ingestURL    string
	metadataURL  string
}

func main() {
	logger := log.New(os.Stdout, "[GATEWAY] ", log.LstdFlags|log.Lshortfile)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET environment variable is required")
	}

	gw := &Gateway{
		router:      mux.NewRouter(),
		logger:      logger,
		jwtManager:  auth.NewJWTManager(jwtSecret, 24*time.Hour),
		rateLimiter: middleware.NewRateLimiter(100, time.Minute, 100),
		ingestURL:   getEnv("INGEST_URL", "http://localhost:8081"),
		metadataURL: getEnv("METADATA_URL", "http://localhost:8083"),
	}
	gw.authMW = auth.NewAuthMiddleware(gw.jwtManager)

	gw.setupRoutes()

	port := getEnv("PORT", "8080")

	// CORS
	allowedOrigins := []string{"http://localhost:3000"}
	if frontendURL := getEnv("FRONTEND_URL", ""); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	if prodURL := getEnv("PRODUCTION_FRONTEND_URL", ""); prodURL != "" {
		allowedOrigins = append(allowedOrigins, prodURL)
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      c.Handler(gw.router),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Printf("Gateway listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Println("Shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logger.Println("Gateway stopped")
}

func (gw *Gateway) setupRoutes() {
	// Health check (public)
	gw.router.HandleFunc("/api/v1/health", gw.healthCheck).Methods("GET")

	// Auth endpoints (public, rate limited)
	authRouter := gw.router.PathPrefix("/api/v1/auth").Subrouter()
	authRouter.Use(gw.rateLimiter.Limit)
	authRouter.HandleFunc("/register", gw.proxyToMetadata).Methods("POST")
	authRouter.HandleFunc("/login", gw.proxyToMetadata).Methods("POST")

	// Protected routes
	protected := gw.router.PathPrefix("/api/v1").Subrouter()
	protected.Use(gw.authMW.Authenticate)

	// File operations → Ingest service
	protected.HandleFunc("/files/upload", gw.proxyToIngest).Methods("POST")
	protected.HandleFunc("/files/{fileId}/download", gw.proxyToIngest).Methods("GET")

	// Metadata operations → Metadata service
	protected.HandleFunc("/files", gw.proxyToMetadata).Methods("GET")
	protected.HandleFunc("/files/{fileId}", gw.proxyToMetadata).Methods("DELETE")
	protected.HandleFunc("/auth/profile", gw.proxyToMetadata).Methods("GET")
}

func (gw *Gateway) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "echofs-gateway",
	})
}

func (gw *Gateway) proxyToIngest(w http.ResponseWriter, r *http.Request) {
	gw.reverseProxy(w, r, gw.ingestURL)
}

func (gw *Gateway) proxyToMetadata(w http.ResponseWriter, r *http.Request) {
	gw.reverseProxy(w, r, gw.metadataURL)
}

func (gw *Gateway) reverseProxy(w http.ResponseWriter, r *http.Request, targetURL string) {
	target, err := url.Parse(targetURL)
	if err != nil {
		gw.logger.Printf("Invalid proxy target: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		gw.logger.Printf("Proxy error to %s: %v", targetURL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Service unavailable",
		})
	}

	// Forward auth context as headers for downstream services
	if claims, ok := auth.GetUserFromContext(r.Context()); ok {
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-User-Email", claims.Email)
		r.Header.Set("X-User-Role", claims.Role)
	}

	// Strip /api/v1 prefix if downstream expects different paths
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "")
	r.Host = target.Host

	proxy.ServeHTTP(w, r)
}

// Ensure io and fmt are used (they're needed for potential request body forwarding)
var _ = io.Copy
var _ = fmt.Sprintf
