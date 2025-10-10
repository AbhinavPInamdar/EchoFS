package metrics

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type MetricsDashboard struct {
	FileOperations struct {
		TotalUploads   float64 `json:"total_uploads"`
		TotalDownloads float64 `json:"total_downloads"`
		TotalDeletes   float64 `json:"total_deletes"`
	} `json:"file_operations"`
	
	Performance struct {
		AvgUploadTime   float64 `json:"avg_upload_time_seconds"`
		AvgDownloadTime float64 `json:"avg_download_time_seconds"`
	} `json:"performance"`
	
	System struct {
		ActiveConnections int     `json:"active_connections"`
		StorageUsage      float64 `json:"storage_usage_bytes"`
	} `json:"system"`
	
	GRPC struct {
		TotalRequests float64 `json:"total_requests"`
		TotalErrors   float64 `json:"total_errors"`
	} `json:"grpc"`
	
	HTTP struct {
		TotalRequests float64 `json:"total_requests"`
		TotalErrors   float64 `json:"total_errors"`
	} `json:"http"`
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if AppMetrics == nil {
		http.Error(w, "Metrics not initialized", http.StatusInternalServerError)
		return
	}
	
	dashboard := MetricsDashboard{}
	
	// Get file operation metrics
	dashboard.FileOperations.TotalUploads = getCounterValue(AppMetrics.FileUploadsTotal)
	dashboard.FileOperations.TotalDownloads = getCounterValue(AppMetrics.FileDownloadsTotal)
	dashboard.FileOperations.TotalDeletes = getCounterValue(AppMetrics.FileDeletesTotal)
	
	// Get performance metrics (averages from histograms)
	dashboard.Performance.AvgUploadTime = getHistogramMean(AppMetrics.UploadDuration)
	dashboard.Performance.AvgDownloadTime = getHistogramMean(AppMetrics.DownloadDuration)
	
	// Get system metrics
	dashboard.System.ActiveConnections = int(getGaugeValue(AppMetrics.ActiveConnections))
	dashboard.System.StorageUsage = getGaugeValue(AppMetrics.StorageUsageBytes)
	
	// Get gRPC metrics
	dashboard.GRPC.TotalRequests = getCounterVecSum(AppMetrics.GRPCRequestsTotal)
	dashboard.GRPC.TotalErrors = getCounterVecSum(AppMetrics.GRPCErrors)
	
	// Get HTTP metrics
	dashboard.HTTP.TotalRequests = getCounterVecSum(HTTPRequestsTotal)
	dashboard.HTTP.TotalErrors = getCounterVecSum(HTTPErrors)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

func getCounterValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	counter.Write(metric)
	return metric.GetCounter().GetValue()
}

func getGaugeValue(gauge prometheus.Gauge) float64 {
	metric := &dto.Metric{}
	gauge.Write(metric)
	return metric.GetGauge().GetValue()
}

func getHistogramMean(histogram prometheus.Histogram) float64 {
	metric := &dto.Metric{}
	histogram.Write(metric)
	h := metric.GetHistogram()
	if h.GetSampleCount() == 0 {
		return 0
	}
	return h.GetSampleSum() / float64(h.GetSampleCount())
}

func getCounterVecSum(counterVec *prometheus.CounterVec) float64 {
	metricFamilies, _ := prometheus.DefaultGatherer.Gather()
	for _, mf := range metricFamilies {
		if mf.GetName() == getMetricName(counterVec) {
			sum := 0.0
			for _, metric := range mf.GetMetric() {
				sum += metric.GetCounter().GetValue()
			}
			return sum
		}
	}
	return 0
}

func getMetricName(metric prometheus.Collector) string {
	desc := make(chan *prometheus.Desc, 1)
	metric.Describe(desc)
	close(desc)
	for d := range desc {
		return d.String()
	}
	return ""
}