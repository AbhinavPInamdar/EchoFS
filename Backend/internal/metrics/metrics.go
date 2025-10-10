package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusClient interface for metrics operations
type PrometheusClient interface {
	Query(query string) (interface{}, error)
	QueryRange(query string, start, end time.Time, step time.Duration) (interface{}, error)
}

// MockPrometheusClient for testing
type MockPrometheusClient struct{}

func (m *MockPrometheusClient) Query(query string) (interface{}, error) {
	return map[string]interface{}{"status": "success", "data": []interface{}{}}, nil
}

func (m *MockPrometheusClient) QueryRange(query string, start, end time.Time, step time.Duration) (interface{}, error) {
	return map[string]interface{}{"status": "success", "data": []interface{}{}}, nil
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(endpoint string) PrometheusClient {
	// For now, return a mock client
	// In a real implementation, this would create an actual Prometheus client
	return &MockPrometheusClient{}
}

type Metrics struct {
	// File operation metrics
	FileUploadsTotal     prometheus.Counter
	FileDownloadsTotal   prometheus.Counter
	FileDeletesTotal     prometheus.Counter
	FileOperationErrors  *prometheus.CounterVec
	
	// File size metrics
	FileSizeBytes        prometheus.Histogram
	ChunkSizeBytes       prometheus.Histogram
	
	// Performance metrics
	UploadDuration       prometheus.Histogram
	DownloadDuration     prometheus.Histogram
	ChunkProcessingTime  prometheus.Histogram
	
	// System metrics
	ActiveConnections    prometheus.Gauge
	WorkerHealthStatus   *prometheus.GaugeVec
	StorageUsageBytes    prometheus.Gauge
	
	// gRPC metrics
	GRPCRequestsTotal    *prometheus.CounterVec
	GRPCRequestDuration  *prometheus.HistogramVec
	GRPCErrors           *prometheus.CounterVec

	// Consistency and replication metrics
	ObjectModeChanges    *prometheus.CounterVec
	ReplicationLatency   *prometheus.HistogramVec
	QuorumFailures       *prometheus.CounterVec
	AsyncQueueSize       prometheus.Gauge
	NodeHealthStatus     *prometheus.GaugeVec
	ConsistencyModeGauge *prometheus.GaugeVec
	
	// S3 metrics
	S3OperationsTotal    *prometheus.CounterVec
	S3OperationDuration  *prometheus.HistogramVec
	S3Errors             *prometheus.CounterVec
	
	// User metrics
	ActiveUsers          prometheus.Gauge
	UserFileCount        *prometheus.GaugeVec
	UserStorageUsage     *prometheus.GaugeVec
}

var AppMetrics *Metrics

func InitMetrics() *Metrics {
	metrics := &Metrics{
		// File operation counters
		FileUploadsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "echofs_file_uploads_total",
			Help: "Total number of file uploads",
		}),
		
		FileDownloadsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "echofs_file_downloads_total",
			Help: "Total number of file downloads",
		}),
		
		FileDeletesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "echofs_file_deletes_total",
			Help: "Total number of file deletions",
		}),
		
		FileOperationErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_file_operation_errors_total",
			Help: "Total number of file operation errors",
		}, []string{"operation", "error_type"}),
		
		// File size histograms
		FileSizeBytes: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "echofs_file_size_bytes",
			Help: "Size of uploaded files in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to ~1GB
		}),
		
		ChunkSizeBytes: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "echofs_chunk_size_bytes",
			Help: "Size of file chunks in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to ~32MB
		}),
		
		// Performance histograms
		UploadDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "echofs_upload_duration_seconds",
			Help: "Time taken to upload files",
			Buckets: prometheus.DefBuckets,
		}),
		
		DownloadDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "echofs_download_duration_seconds",
			Help: "Time taken to download files",
			Buckets: prometheus.DefBuckets,
		}),
		
		ChunkProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "echofs_chunk_processing_seconds",
			Help: "Time taken to process individual chunks",
			Buckets: prometheus.DefBuckets,
		}),
		
		// System gauges
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "echofs_active_connections",
			Help: "Number of active connections",
		}),
		
		WorkerHealthStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "echofs_worker_health_status",
			Help: "Health status of workers (1=healthy, 0=unhealthy)",
		}, []string{"worker_id", "worker_address"}),
		
		StorageUsageBytes: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "echofs_storage_usage_bytes",
			Help: "Total storage usage in bytes",
		}),
		
		// gRPC metrics
		GRPCRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_grpc_requests_total",
			Help: "Total number of gRPC requests",
		}, []string{"method", "status"}),
		
		GRPCRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "echofs_grpc_request_duration_seconds",
			Help: "Duration of gRPC requests",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		
		GRPCErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_grpc_errors_total",
			Help: "Total number of gRPC errors",
		}, []string{"method", "error_code"}),
		
		// S3 metrics
		S3OperationsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_s3_operations_total",
			Help: "Total number of S3 operations",
		}, []string{"operation", "status"}),
		
		S3OperationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "echofs_s3_operation_duration_seconds",
			Help: "Duration of S3 operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		
		S3Errors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_s3_errors_total",
			Help: "Total number of S3 errors",
		}, []string{"operation", "error_type"}),

		// Consistency and replication metrics
		ObjectModeChanges: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_object_mode_changes_total",
			Help: "Total number of object consistency mode changes",
		}, []string{"object_id", "from_mode", "to_mode", "reason"}),

		ReplicationLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "echofs_replication_latency_seconds",
			Help:    "Latency of replication operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		}, []string{"strategy", "operation"}),

		QuorumFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "echofs_quorum_failures_total",
			Help: "Total number of quorum write failures",
		}, []string{"reason"}),

		AsyncQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "echofs_async_queue_size",
			Help: "Current size of async replication queue",
		}),

		NodeHealthStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "echofs_node_health_status",
			Help: "Health status of worker nodes (1=healthy, 0=unhealthy)",
		}, []string{"node_id"}),

		ConsistencyModeGauge: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "echofs_consistency_mode",
			Help: "Current consistency mode for objects (0=Strong, 1=Available, 2=Hybrid)",
		}, []string{"object_id"}),
		
		// User metrics
		ActiveUsers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "echofs_active_users",
			Help: "Number of active users",
		}),
		
		UserFileCount: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "echofs_user_file_count",
			Help: "Number of files per user",
		}, []string{"user_id"}),
		
		UserStorageUsage: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "echofs_user_storage_usage_bytes",
			Help: "Storage usage per user in bytes",
		}, []string{"user_id"}),
	}
	
	AppMetrics = metrics
	return metrics
}

// Helper functions for common operations
func (m *Metrics) RecordFileUpload(fileSize int64, duration time.Duration) {
	m.FileUploadsTotal.Inc()
	m.FileSizeBytes.Observe(float64(fileSize))
	m.UploadDuration.Observe(duration.Seconds())
}

func (m *Metrics) RecordFileDownload(duration time.Duration) {
	m.FileDownloadsTotal.Inc()
	m.DownloadDuration.Observe(duration.Seconds())
}

func (m *Metrics) RecordFileDelete() {
	m.FileDeletesTotal.Inc()
}

func (m *Metrics) RecordFileError(operation, errorType string) {
	m.FileOperationErrors.WithLabelValues(operation, errorType).Inc()
}

func (m *Metrics) RecordChunkProcessing(chunkSize int64, duration time.Duration) {
	m.ChunkSizeBytes.Observe(float64(chunkSize))
	m.ChunkProcessingTime.Observe(duration.Seconds())
}

func (m *Metrics) UpdateWorkerHealth(workerID, address string, healthy bool) {
	status := 0.0
	if healthy {
		status = 1.0
	}
	m.WorkerHealthStatus.WithLabelValues(workerID, address).Set(status)
}

func (m *Metrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

func (m *Metrics) RecordGRPCError(method, errorCode string) {
	m.GRPCErrors.WithLabelValues(method, errorCode).Inc()
}

func (m *Metrics) RecordS3Operation(operation, status string, duration time.Duration) {
	m.S3OperationsTotal.WithLabelValues(operation, status).Inc()
	m.S3OperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (m *Metrics) RecordS3Error(operation, errorType string) {
	m.S3Errors.WithLabelValues(operation, errorType).Inc()
}

func (m *Metrics) UpdateUserMetrics(userID string, fileCount int, storageUsage int64) {
	m.UserFileCount.WithLabelValues(userID).Set(float64(fileCount))
	m.UserStorageUsage.WithLabelValues(userID).Set(float64(storageUsage))
}