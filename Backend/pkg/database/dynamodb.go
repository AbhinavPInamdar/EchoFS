package database

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBService struct {
	client *dynamodb.Client
	tables struct {
		Files    string
		Chunks   string
		Sessions string
	}
}

type FileMetadata struct {
	FileID          string    `dynamodbav:"file_id"`
	FileName        string    `dynamodbav:"file_name"`
	FileSize        int64     `dynamodbav:"file_size"`
	UploadedBy      string    `dynamodbav:"uploaded_by"`
	CreatedAt       time.Time `dynamodbav:"created_at"`
	UpdatedAt       time.Time `dynamodbav:"updated_at"`
	Status          string    `dynamodbav:"status"`
	ChunkCount      int       `dynamodbav:"chunk_count"`
	CompressionType string    `dynamodbav:"compression_type"`
	S3BucketName    string    `dynamodbav:"s3_bucket_name"`
}

type ChunkMetadata struct {
	FileID     string    `dynamodbav:"file_id"`
	ChunkIndex int       `dynamodbav:"chunk_index"`
	ChunkID    string    `dynamodbav:"chunk_id"`
	S3Key      string    `dynamodbav:"s3_key"`
	ChunkSize  int64     `dynamodbav:"chunk_size"`
	MD5Hash    string    `dynamodbav:"md5_hash"`
	WorkerNodes []string `dynamodbav:"worker_nodes"`
	CreatedAt  time.Time `dynamodbav:"created_at"`
}

type UploadSession struct {
	SessionID        string    `dynamodbav:"session_id"`
	UserID           string    `dynamodbav:"user_id"`
	FileName         string    `dynamodbav:"file_name"`
	FileSize         int64     `dynamodbav:"file_size"`
	Status           string    `dynamodbav:"status"`
	CreatedAt        time.Time `dynamodbav:"created_at"`
	ExpiresAt        time.Time `dynamodbav:"expires_at"`
	ChunksCompleted  int       `dynamodbav:"chunks_completed"`
	TotalChunks      int       `dynamodbav:"total_chunks"`
}

func NewDynamoDBService(client *dynamodb.Client, filesTable, chunksTable, sessionsTable string) *DynamoDBService {
	return &DynamoDBService{
		client: client,
		tables: struct {
			Files    string
			Chunks   string
			Sessions string
		}{
			Files:    filesTable,
			Chunks:   chunksTable,
			Sessions: sessionsTable,
		},
	}
}

func (d *DynamoDBService) CreateTables(ctx context.Context) error {

	err := d.createFilesTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create files table: %w", err)
	}

	err = d.createChunksTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create chunks table: %w", err)
	}

	err = d.createSessionsTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	return nil
}

func (d *DynamoDBService) CreateFile(ctx context.Context, file *FileMetadata) error {
	item, err := attributevalue.MarshalMap(file)
	if err != nil {
		return fmt.Errorf("failed to marshal file metadata: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tables.Files),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}

	return nil
}

func (d *DynamoDBService) GetFile(ctx context.Context, fileID string) (*FileMetadata, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tables.Files),
		Key: map[string]types.AttributeValue{
			"file_id": &types.AttributeValueMemberS{Value: fileID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}

	var file FileMetadata
	err = attributevalue.UnmarshalMap(result.Item, &file)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file: %w", err)
	}

	return &file, nil
}

func (d *DynamoDBService) ListFiles(ctx context.Context, userID string) ([]*FileMetadata, error) {
	var files []*FileMetadata

	if userID != "" {

		result, err := d.client.Scan(ctx, &dynamodb.ScanInput{
			TableName:        aws.String(d.tables.Files),
			FilterExpression: aws.String("uploaded_by = :user_id"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":user_id": &types.AttributeValueMemberS{Value: userID},
			},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list files for user: %w", err)
		}

		for _, item := range result.Items {
			var file FileMetadata
			err = attributevalue.UnmarshalMap(item, &file)
			if err != nil {
				continue
			}
			files = append(files, &file)
		}
	} else {

		result, err := d.client.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(d.tables.Files),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list all files: %w", err)
		}

		for _, item := range result.Items {
			var file FileMetadata
			err = attributevalue.UnmarshalMap(item, &file)
			if err != nil {
				continue
			}
			files = append(files, &file)
		}
	}

	return files, nil
}

func (d *DynamoDBService) CreateChunk(ctx context.Context, chunk *ChunkMetadata) error {
	item, err := attributevalue.MarshalMap(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk metadata: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tables.Chunks),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to create chunk record: %w", err)
	}

	return nil
}

func (d *DynamoDBService) GetChunksForFile(ctx context.Context, fileID string) ([]*ChunkMetadata, error) {
	result, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tables.Chunks),
		KeyConditionExpression: aws.String("file_id = :file_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":file_id": &types.AttributeValueMemberS{Value: fileID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get chunks for file: %w", err)
	}

	var chunks []*ChunkMetadata
	for _, item := range result.Items {
		var chunk ChunkMetadata
		err = attributevalue.UnmarshalMap(item, &chunk)
		if err != nil {
			continue
		}
		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

func (d *DynamoDBService) CreateSession(ctx context.Context, session *UploadSession) error {
	item, err := attributevalue.MarshalMap(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tables.Sessions),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (d *DynamoDBService) GetSession(ctx context.Context, sessionID string) (*UploadSession, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tables.Sessions),
		Key: map[string]types.AttributeValue{
			"session_id": &types.AttributeValueMemberS{Value: sessionID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	var session UploadSession
	err = attributevalue.UnmarshalMap(result.Item, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (d *DynamoDBService) createFilesTable(ctx context.Context) error {
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(d.tables.Files),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("file_id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("file_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {

		if _, ok := err.(*types.ResourceInUseException); ok {
			return nil
		}
		return err
	}

	return nil
}

func (d *DynamoDBService) createChunksTable(ctx context.Context) error {
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(d.tables.Chunks),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("file_id"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("chunk_index"),
				KeyType:       types.KeyTypeRange,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("file_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("chunk_index"),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {

		if _, ok := err.(*types.ResourceInUseException); ok {
			return nil
		}
		return err
	}

	return nil
}

func (d *DynamoDBService) createSessionsTable(ctx context.Context) error {
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(d.tables.Sessions),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("session_id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("session_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {

		if _, ok := err.(*types.ResourceInUseException); ok {
			return nil
		}
		return err
	}

	return nil
}