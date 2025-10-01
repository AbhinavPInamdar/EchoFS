package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// AWSConfig holds AWS service clients and configuration
type AWSConfig struct {
	Config         aws.Config
	RDSClient      *rds.Client
	ElastiCache    *elasticache.Client
	CloudWatch     *cloudwatch.Client
	Region         string
	DatabaseURL    string
	RedisEndpoint  string
}

// NewAWSConfig creates a new AWS configuration with all required clients
func NewAWSConfig(ctx context.Context, region, databaseURL, redisEndpoint string) (*AWSConfig, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create service clients
	rdsClient := rds.NewFromConfig(cfg)
	elastiCacheClient := elasticache.NewFromConfig(cfg)
	cloudWatchClient := cloudwatch.NewFromConfig(cfg)

	return &AWSConfig{
		Config:        cfg,
		RDSClient:     rdsClient,
		ElastiCache:   elastiCacheClient,
		CloudWatch:    cloudWatchClient,
		Region:        region,
		DatabaseURL:   databaseURL,
		RedisEndpoint: redisEndpoint,
	}, nil
}

// ValidateAWSServices validates that AWS services are accessible
func (a *AWSConfig) ValidateAWSServices(ctx context.Context) error {
	// Test RDS connectivity by describing DB instances
	_, err := a.RDSClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RDS: %w", err)
	}

	// Test ElastiCache connectivity
	_, err = a.ElastiCache.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ElastiCache: %w", err)
	}

	// Test CloudWatch connectivity
	_, err = a.CloudWatch.ListMetrics(ctx, &cloudwatch.ListMetricsInput{
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to CloudWatch: %w", err)
	}

	return nil
}