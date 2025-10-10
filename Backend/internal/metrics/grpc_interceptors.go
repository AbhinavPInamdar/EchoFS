package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor for metrics collection
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Record metrics
		if AppMetrics != nil {
			duration := time.Since(start)
			method := info.FullMethod
			
			// Determine status
			grpcStatus := "success"
			errorCode := "none"
			
			if err != nil {
				grpcStatus = "error"
				if s, ok := status.FromError(err); ok {
					errorCode = s.Code().String()
				} else {
					errorCode = "unknown"
				}
				AppMetrics.RecordGRPCError(method, errorCode)
			}
			
			AppMetrics.RecordGRPCRequest(method, grpcStatus, duration)
		}
		
		return resp, err
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor for metrics collection
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		
		// Call the method
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		// Record metrics
		if AppMetrics != nil {
			duration := time.Since(start)
			
			// Determine status
			grpcStatus := "success"
			errorCode := "none"
			
			if err != nil {
				grpcStatus = "error"
				if s, ok := status.FromError(err); ok {
					errorCode = s.Code().String()
				} else {
					errorCode = "unknown"
				}
				AppMetrics.RecordGRPCError(method, errorCode)
			}
			
			AppMetrics.RecordGRPCRequest(method, grpcStatus, duration)
		}
		
		return err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for metrics collection
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		
		// Call the handler
		err := handler(srv, stream)
		
		// Record metrics
		if AppMetrics != nil {
			duration := time.Since(start)
			method := info.FullMethod
			
			// Determine status
			grpcStatus := "success"
			errorCode := "none"
			
			if err != nil {
				grpcStatus = "error"
				if s, ok := status.FromError(err); ok {
					errorCode = s.Code().String()
				} else {
					errorCode = "unknown"
				}
				AppMetrics.RecordGRPCError(method, errorCode)
			}
			
			AppMetrics.RecordGRPCRequest(method, grpcStatus, duration)
		}
		
		return err
	}
}