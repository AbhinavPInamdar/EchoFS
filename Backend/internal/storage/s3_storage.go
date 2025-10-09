package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)


type S3Storage struct {
	client     *s3.Client
	bucketName string
}


func NewS3Storage(client *s3.Client, bucketName string) *S3Storage {
	return &S3Storage{
		client:     client,
		bucketName: bucketName,
	}
}


func (s *S3Storage) StoreChunk(ctx context.Context, fileID, chunkID string, chunkIndex int, data []byte) error {
	key := s.generateChunkKey(fileID, chunkID, chunkIndex)
	
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
		Metadata: map[string]string{
			"file-id":     fileID,
			"chunk-id":    chunkID,
			"chunk-index": fmt.Sprintf("%d", chunkIndex),
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to store chunk %s: %w", chunkID, err)
	}
	
	return nil
}


func (s *S3Storage) RetrieveChunk(ctx context.Context, fileID, chunkID string, chunkIndex int) ([]byte, error) {
	key := s.generateChunkKey(fileID, chunkID, chunkIndex)
	
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunk %s: %w", chunkID, err)
	}
	defer result.Body.Close()
	
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk data: %w", err)
	}
	
	return data, nil
}


func (s *S3Storage) DeleteChunk(ctx context.Context, fileID, chunkID string, chunkIndex int) error {
	key := s.generateChunkKey(fileID, chunkID, chunkIndex)
	
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete chunk %s: %w", chunkID, err)
	}
	
	return nil
}


func (s *S3Storage) ListChunks(ctx context.Context, fileID string) ([]string, error) {
	prefix := fmt.Sprintf("files/%s/chunks/", fileID)
	
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list chunks for file %s: %w", fileID, err)
	}
	
	var chunks []string
	for _, obj := range result.Contents {
		if obj.Key != nil {
			chunks = append(chunks, *obj.Key)
		}
	}
	
	return chunks, nil
}


func (s *S3Storage) DeleteAllChunks(ctx context.Context, fileID string) error {
	chunks, err := s.ListChunks(ctx, fileID)
	if err != nil {
		return err
	}
	
	if len(chunks) == 0 {
		return nil
	}
	

	var objects []types.ObjectIdentifier
	for _, chunk := range chunks {
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(chunk),
		})
	}
	
	_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{
			Objects: objects,
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete chunks for file %s: %w", fileID, err)
	}
	
	return nil
}


func (s *S3Storage) ChunkExists(ctx context.Context, fileID, chunkID string, chunkIndex int) (bool, error) {
	key := s.generateChunkKey(fileID, chunkID, chunkIndex)
	
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	
	if err != nil {

		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check chunk existence: %w", err)
	}
	
	return true, nil
}


func (s *S3Storage) generateChunkKey(fileID, chunkID string, chunkIndex int) string {
	return fmt.Sprintf("files/%s/chunks/%s_%d", fileID, chunkID, chunkIndex)
}


func (s *S3Storage) EnsureBucket(ctx context.Context) error {

	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	
	if err == nil {
		return nil
	}
	

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", s.bucketName, err)
	}
	
	return nil
}



