package aws

import (
	"context"
	"fmt"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type AWSConfig struct {
	Config         aws.Config
	RDSClient      *rds.Client
	ElastiCache    *elasticache.Client
	CloudWatch     *cloudwatch.Client
	S3			   *s3.Client
	DynamoDB       *dynamodb.Client
	Region         string
	DatabaseURL    string
	RedisEndpoint  string
	S3BucketName   string
	DynamoDBTables struct {
		Files    string
		Chunks   string
		Sessions string
	}
}

func NewAWSConfig(ctx context.Context, region, databaseURL, redisEndpoint string) (*AWSConfig, error) {

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	dynamodbClient := dynamodb.NewFromConfig(cfg)
	rdsClient := rds.NewFromConfig(cfg)
	elastiCacheClient := elasticache.NewFromConfig(cfg)
	cloudWatchClient := cloudwatch.NewFromConfig(cfg)

	s3BucketName := os.Getenv("S3_BUCKET_NAME")
	if s3BucketName == "" {
		s3BucketName = "echofs-chunks-bucket"
	}

	filesTable := os.Getenv("DYNAMODB_FILES_TABLE")
	if filesTable == "" {
		filesTable = "echofs-files"
	}
	chunksTable := os.Getenv("DYNAMODB_CHUNKS_TABLE")
	if chunksTable == "" {
		chunksTable = "echofs-chunks"
	}
	sessionsTable := os.Getenv("DYNAMODB_SESSIONS_TABLE")
	if sessionsTable == "" {
		sessionsTable = "echofs-sessions"
	}

	return &AWSConfig{
		Config:        cfg,
		RDSClient:     rdsClient,
		ElastiCache:   elastiCacheClient,
		CloudWatch:    cloudWatchClient,
		S3:			   s3Client,
		DynamoDB:      dynamodbClient,
		Region:        region,
		DatabaseURL:   databaseURL,
		RedisEndpoint: redisEndpoint,
		S3BucketName:  s3BucketName,
		DynamoDBTables: struct {
			Files    string
			Chunks   string
			Sessions string
		}{
			Files:    filesTable,
			Chunks:   chunksTable,
			Sessions: sessionsTable,
		},
	}, nil
}

func (a *AWSConfig) ValidateAWSServices(ctx context.Context) error {

	_, err := a.RDSClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RDS: %w", err)
	}

	_, err = a.ElastiCache.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		MaxRecords: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ElastiCache: %w", err)
	}

	_, err = a.CloudWatch.ListMetrics(ctx, &cloudwatch.ListMetricsInput{})
	if err != nil {
		return fmt.Errorf("failed to connect to CloudWatch: %w", err)
	}

	_, err = a.DynamoDB.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return fmt.Errorf("Failed to connect to DynamoDB: %w", err)
	}

	_, err = a.S3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("Failed to connect to S3: %w", err)
	}
	return nil
}