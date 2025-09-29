package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"echofs-backend/pkg/auth"
)

// LoggingInterceptor logs gRPC calls
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Log the call
		duration := time.Since(start)
		if err != nil {
			log.Printf("gRPC call %s failed in %v: %v", info.FullMethod, duration, err)
		} else {
			log.Printf("gRPC call %s completed in %v", info.FullMethod, duration)
		}
		
		return resp, err
	}
}

// StreamLoggingInterceptor logs streaming gRPC calls
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		// Call the handler
		err := handler(srv, stream)
		
		// Log the call
		duration := time.Since(start)
		if err != nil {
			log.Printf("gRPC stream %s failed in %v: %v", info.FullMethod, duration, err)
		} else {
			log.Printf("gRPC stream %s completed in %v", info.FullMethod, duration)
		}
		
		return err
	}
}

// AuthInterceptor validates JWT tokens
func AuthInterceptor(jwtManager *auth.JWTManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for health checks
		if info.FullMethod == "/echofs.v1.EchoFSService/HealthCheck" {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata not found")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token not found")
		}

		token := values[0]
		if len(token) < 7 || token[:7] != "Bearer " {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token[7:])
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add claims to context
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "role", claims.Role)

		return handler(ctx, req)
	}
}

// StreamAuthInterceptor validates JWT tokens for streaming calls
func StreamAuthInterceptor(jwtManager *auth.JWTManager) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for health checks
		if info.FullMethod == "/echofs.v1.EchoFSService/HealthCheck" {
			return handler(srv, stream)
		}

		// Extract token from metadata
		ctx := stream.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "metadata not found")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return status.Errorf(codes.Unauthenticated, "authorization token not found")
		}

		token := values[0]
		if len(token) < 7 || token[:7] != "Bearer " {
			return status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token[7:])
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Create new context with claims
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "role", claims.Role)

		// Wrap the stream with new context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream wraps grpc.ServerStream to override context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// RecoveryInterceptor recovers from panics
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in %s: %v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in streaming calls
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in stream %s: %v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, stream)
	}
}

// RateLimitInterceptor implements basic rate limiting
func RateLimitInterceptor(maxRequests int, window time.Duration) grpc.UnaryServerInterceptor {
	requestCounts := make(map[string]int)
	lastReset := time.Now()
	
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract client identifier (could be IP, user ID, etc.)
		clientID := getClientID(ctx)
		
		// Reset counts if window has passed
		if time.Since(lastReset) > window {
			requestCounts = make(map[string]int)
			lastReset = time.Now()
		}
		
		// Check rate limit
		requestCounts[clientID]++
		if requestCounts[clientID] > maxRequests {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}
		
		return handler(ctx, req)
	}
}

// getClientID extracts client identifier from context
func getClientID(ctx context.Context) string {
	// Try to get user ID first
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		return userID
	}
	
	// Fall back to peer address
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}
	
	if values := md["x-forwarded-for"]; len(values) > 0 {
		return values[0]
	}
	
	if values := md["x-real-ip"]; len(values) > 0 {
		return values[0]
	}
	
	return "unknown"
}

// MetricsInterceptor collects metrics for gRPC calls
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Record metrics
		duration := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		
		// TODO: Integrate with Prometheus metrics
		log.Printf("Metrics: method=%s duration=%v status=%s", info.FullMethod, duration, status)
		
		return resp, err
	}
}

// StreamMetricsInterceptor collects metrics for streaming gRPC calls
func StreamMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		// Call the handler
		err := handler(srv, stream)
		
		// Record metrics
		duration := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}
		
		// TODO: Integrate with Prometheus metrics
		log.Printf("Stream Metrics: method=%s duration=%v status=%s", info.FullMethod, duration, status)
		
		return err
	}
}

// GetUserFromContext extracts user information from context
func GetUserFromContext(ctx context.Context) (userID, username, email, role string) {
	if uid, ok := ctx.Value("user_id").(string); ok {
		userID = uid
	}
	if uname, ok := ctx.Value("username").(string); ok {
		username = uname
	}
	if em, ok := ctx.Value("email").(string); ok {
		email = em
	}
	if r, ok := ctx.Value("role").(string); ok {
		role = r
	}
	return
}


