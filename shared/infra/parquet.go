package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/parquet-go/parquet-go"
)

// FileMetadata represents file metadata stored in Parquet
type FileMetadata struct {
	FileHash     string    `parquet:"file_hash,snappy"`
	FilePath     string    `parquet:"file_path,snappy"`
	BucketName   string    `parquet:"bucket_name,snappy"`
	OriginalName string    `parquet:"original_name,snappy"`
	ContentType  string    `parquet:"content_type,snappy"`
	FileSize     int64     `parquet:"file_size"`
	UploadedAt   time.Time `parquet:"uploaded_at"`
}

type ParquetService struct {
	minioClient    *MinioClient
	metadataBucket string
	metadataFile   string
}

func NewParquetService(minioClient *MinioClient) *ParquetService {
	return &ParquetService{
		minioClient:    minioClient,
		metadataBucket: "metadata",
		metadataFile:   "files-metadata.parquet",
	}
}

// LoadMetadata loads all file metadata from Parquet file
func (ps *ParquetService) LoadMetadata(ctx context.Context) ([]FileMetadata, error) {
	// Ensure metadata bucket exists
	if err := ps.minioClient.EnsureBucketByName(ctx, ps.metadataBucket); err != nil {
		return nil, fmt.Errorf("failed to ensure metadata bucket: %w", err)
	}

	// Try to download existing metadata file
	data, _, err := ps.minioClient.GetObjectFromBucket(ctx, ps.metadataBucket, ps.metadataFile)
	if err != nil {
		// If file doesn't exist, return empty slice
		return []FileMetadata{}, nil
	}

	// Read Parquet file
	reader := bytes.NewReader(data)
	parquetReader := parquet.NewGenericReader[FileMetadata](reader)
	defer parquetReader.Close()

	var metadata []FileMetadata
	rows := make([]FileMetadata, 1000) // Read in batches
	for {
		n, err := parquetReader.Read(rows)
		if n > 0 {
			metadata = append(metadata, rows[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read parquet: %w", err)
		}
	}

	return metadata, nil
}

// SaveMetadata saves all file metadata to Parquet file
func (ps *ParquetService) SaveMetadata(ctx context.Context, metadata []FileMetadata) error {
	// Ensure metadata bucket exists
	if err := ps.minioClient.EnsureBucketByName(ctx, ps.metadataBucket); err != nil {
		return fmt.Errorf("failed to ensure metadata bucket: %w", err)
	}

	// Write to Parquet buffer
	buf := new(bytes.Buffer)
	parquetWriter := parquet.NewGenericWriter[FileMetadata](buf, parquet.Compression(&parquet.Snappy))

	_, err := parquetWriter.Write(metadata)
	if err != nil {
		return fmt.Errorf("failed to write parquet: %w", err)
	}

	if err := parquetWriter.Close(); err != nil {
		return fmt.Errorf("failed to close parquet writer: %w", err)
	}

	// Upload to MinIO
	_, err = ps.minioClient.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(ps.metadataBucket),
		Key:         aws.String(ps.metadataFile),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	return nil
}

// CheckFileByHash checks if a file with the given hash exists using Parquet metadata
func (ps *ParquetService) CheckFileByHash(ctx context.Context, bucket, hash string) (string, bool, error) {
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return "", false, err
	}

	// Search for matching hash in the same bucket
	for _, item := range metadata {
		if item.FileHash == hash && item.BucketName == bucket {
			return item.FilePath, true, nil
		}
	}

	return "", false, nil
}

// AddFileMetadata adds a new file metadata entry
func (ps *ParquetService) AddFileMetadata(ctx context.Context, meta FileMetadata) error {
	// Load existing metadata
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return err
	}

	// Check if hash already exists (to avoid duplicates in metadata)
	for i, item := range metadata {
		if item.FileHash == meta.FileHash && item.BucketName == meta.BucketName {
			// Update existing entry
			metadata[i] = meta
			return ps.SaveMetadata(ctx, metadata)
		}
	}

	// Add new entry
	metadata = append(metadata, meta)
	return ps.SaveMetadata(ctx, metadata)
}

// RemoveFileMetadata removes a file metadata entry by path
func (ps *ParquetService) RemoveFileMetadata(ctx context.Context, bucket, filePath string) error {
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return err
	}

	// Filter out the file
	var newMetadata []FileMetadata
	for _, item := range metadata {
		if !(item.FilePath == filePath && item.BucketName == bucket) {
			newMetadata = append(newMetadata, item)
		}
	}

	return ps.SaveMetadata(ctx, newMetadata)
}

// GetStatistics returns statistics about stored files
func (ps *ParquetService) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_files": len(metadata),
		"total_size":  int64(0),
		"by_bucket":   make(map[string]int),
		"by_type":     make(map[string]int),
	}

	totalSize := int64(0)
	byBucket := make(map[string]int)
	byType := make(map[string]int)

	for _, item := range metadata {
		totalSize += item.FileSize
		byBucket[item.BucketName]++
		byType[item.ContentType]++
	}

	stats["total_size"] = totalSize
	stats["by_bucket"] = byBucket
	stats["by_type"] = byType

	return stats, nil
}

// SearchByHash searches for all files with a specific hash across all buckets
func (ps *ParquetService) SearchByHash(ctx context.Context, hash string) ([]FileMetadata, error) {
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return nil, err
	}

	var results []FileMetadata
	for _, item := range metadata {
		if item.FileHash == hash {
			results = append(results, item)
		}
	}

	return results, nil
}

// OptimizeMetadata removes orphaned entries (files that no longer exist in MinIO)
func (ps *ParquetService) OptimizeMetadata(ctx context.Context) (int, error) {
	metadata, err := ps.LoadMetadata(ctx)
	if err != nil {
		return 0, err
	}

	var validMetadata []FileMetadata
	removedCount := 0

	for _, item := range metadata {
		// Check if file still exists
		_, err := ps.minioClient.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(item.BucketName),
			Key:    aws.String(item.FilePath),
		})
		if err == nil {
			// File exists, keep metadata
			validMetadata = append(validMetadata, item)
		} else {
			// File doesn't exist, remove from metadata
			removedCount++
		}
	}

	if removedCount > 0 {
		if err := ps.SaveMetadata(ctx, validMetadata); err != nil {
			return 0, err
		}
	}

	return removedCount, nil
}
