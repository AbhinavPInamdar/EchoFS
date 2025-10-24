package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		
		resp, err := handler(ctx, req)
		
		if AppMetrics != nil {
			duration := time.Since(start)
			method := info.FullMethod
			
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
		
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		if AppMetrics != nil {
			duration := time.Since(start)
			
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

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		
		err := handler(srv, stream)
		
		if AppMetrics != nil {
			duration := time.Since(start)
			method := info.FullMethod
			
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