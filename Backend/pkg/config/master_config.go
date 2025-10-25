package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type MasterConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	
	DatabaseURL      string `json:"database_url"`
	DatabaseMaxConns int    `json:"database_max_conns"`
	DatabaseTimeout  time.Duration `json:"database_timeout"`
	

	
	ReplicationFactor     int           `json:"replication_factor"`
	VirtualNodesPerWorker int           `json:"virtual_nodes_per_worker"`
	WorkerHealthTimeout   time.Duration `json:"worker_health_timeout"`
	HeartbeatInterval     time.Duration `json:"heartbeat_interval"`
	
	SessionTimeout      time.Duration `json:"session_timeout"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MaxConcurrentUploads int          `json:"max_concurrent_uploads"`
	
	JWTSecret       string        `json:"jwt_secret"`
	JWTExpiry       time.Duration `json:"jwt_expiry"`
	TLSEnabled      bool          `json:"tls_enabled"`
	TLSCertPath     string        `json:"tls_cert_path"`
	TLSKeyPath      string        `json:"tls_key_path"`
	
	MaxGoroutines     int `json:"max_goroutines"`
	RequestBufferSize int `json:"request_buffer_size"`
	ChunkSize         int `json:"chunk_size"`
	
	MetricsEnabled bool   `json:"metrics_enabled"`
	MetricsPort    int    `json:"metrics_port"`
	HealthPort     int    `json:"health_port"`
}

func LoadMasterConfig() (*MasterConfig, error) {
	config := &MasterConfig{
		Host:                  "0.0.0.0",
		Port:                  8080,
		LogLevel:             "info",
		DatabaseMaxConns:     10,
		DatabaseTimeout:      30 * time.Second,
		ReplicationFactor:    3,
		VirtualNodesPerWorker: 100,
		WorkerHealthTimeout:  90 * time.Second,
		HeartbeatInterval:    30 * time.Second,
		SessionTimeout:       24 * time.Hour,
		CleanupInterval:      1 * time.Hour,
		MaxConcurrentUploads: 100,
		JWTExpiry:           24 * time.Hour,
		TLSEnabled:          false,
		MaxGoroutines:       1000,
		RequestBufferSize:   1024,
		ChunkSize:           1024 * 1024, 
		MetricsEnabled:      true,
		MetricsPort:         9090,
		HealthPort:          8081,
	}
	
	if host := os.Getenv("MASTER_HOST"); host != "" {
		config.Host = host
	}
	
	if port := os.Getenv("MASTER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}
	
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}
	
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}
	
	
	
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWTSecret = jwtSecret
	} else {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	
	if replicationFactor := os.Getenv("REPLICATION_FACTOR"); replicationFactor != "" {
		if rf, err := strconv.Atoi(replicationFactor); err == nil {
			config.ReplicationFactor = rf
		}
	}
	
	if chunkSize := os.Getenv("CHUNK_SIZE"); chunkSize != "" {
		if cs, err := strconv.Atoi(chunkSize); err == nil {
			config.ChunkSize = cs
		}
	}
	
	if tlsEnabled := os.Getenv("TLS_ENABLED"); tlsEnabled == "true" {
		config.TLSEnabled = true
		config.TLSCertPath = os.Getenv("TLS_CERT_PATH")
		config.TLSKeyPath = os.Getenv("TLS_KEY_PATH")
		
		if config.TLSCertPath == "" || config.TLSKeyPath == "" {
			return nil, fmt.Errorf("TLS_CERT_PATH and TLS_KEY_PATH are required when TLS is enabled")
		}
	}
	
	return config, nil
}

func (c *MasterConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	
	if c.ReplicationFactor <= 0 {
		return fmt.Errorf("replication factor must be positive")
	}
	
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive")
	}
	

	
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}
	
	return nil
}