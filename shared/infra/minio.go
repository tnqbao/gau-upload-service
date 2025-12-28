package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/tnqbao/gau-upload-service/shared/config"
)

type MinioClient struct {
	Client *s3.Client
}

func NewMinioClient(cfg *appconfig.EnvConfig) (*MinioClient, error) {
	endpoint := cfg.Minio.Endpoint
	accessKey := cfg.Minio.AccessKey
	secretKey := cfg.Minio.SecretKey
	region := cfg.Minio.Region
	useSSL := cfg.Minio.UseSSL

	if region == "" {
		region = "us-east-1"
	}

	// Create a custom endpoint resolver for MinIO
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, reg string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	// Load AWS configuration with custom endpoint and credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for MinIO: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // MinIO requires path-style addressing
		if !useSSL {
			o.EndpointOptions.DisableHTTPS = true
		}
	})

	return &MinioClient{
		Client: s3Client,
	}, nil
}

// PutObjectWithMetadata uploads an object with custom metadata
func (m *MinioClient) PutObjectWithMetadata(ctx context.Context, bucket, key string, data []byte, contentType string, metadata map[string]string) error {
	// Ensure bucket exists
	if err := m.EnsureBucketByName(ctx, bucket); err != nil {
		return err
	}

	_, err := m.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to put object with metadata: %w", err)
	}
	return nil
}

// PutObjectStreamWithMetadata uploads an object from a stream with custom metadata
func (m *MinioClient) PutObjectStreamWithMetadata(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string, metadata map[string]string) error {
	// Ensure bucket exists
	if err := m.EnsureBucketByName(ctx, bucket); err != nil {
		return err
	}

	_, err := m.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		Metadata:      metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to put object stream with metadata: %w", err)
	}
	return nil
}

// CheckFileExistsByHash checks if a file with the same hash exists in the bucket
func (m *MinioClient) CheckFileExistsByHash(ctx context.Context, bucket, hash string) (string, bool, error) {
	// List all objects in the bucket
	resp, err := m.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to list objects: %w", err)
	}

	// Check each object's metadata for matching hash
	for _, item := range resp.Contents {
		// Get object metadata
		headResp, err := m.Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    item.Key,
		})
		if err != nil {
			continue // Skip if can't read metadata
		}

		// Check if file-hash metadata matches
		if fileHash, ok := headResp.Metadata["file-hash"]; ok && fileHash == hash {
			return aws.ToString(item.Key), true, nil
		}
	}

	return "", false, nil
}

// GetObjectFromBucket retrieves an object from a specific bucket
func (m *MinioClient) GetObjectFromBucket(ctx context.Context, bucket, key string) ([]byte, string, error) {
	resp, err := m.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, "", err
	}

	contentType := aws.ToString(resp.ContentType)
	return buf.Bytes(), contentType, nil
}

// DeleteObjectFromBucket deletes an object from a specific bucket
func (m *MinioClient) DeleteObjectFromBucket(ctx context.Context, bucket, key string) error {
	_, err := m.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// ListObjectsFromBucket lists all objects in a specific bucket with optional prefix
func (m *MinioClient) ListObjectsFromBucket(ctx context.Context, bucket, prefix string) ([]string, error) {
	resp, err := m.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var keys []string
	for _, item := range resp.Contents {
		keys = append(keys, aws.ToString(item.Key))
	}
	return keys, nil
}

// EnsureBucketByName creates a bucket by name if it doesn't exist
func (m *MinioClient) EnsureBucketByName(ctx context.Context, bucket string) error {
	_, err := m.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		// Bucket doesn't exist, create it
		_, err = m.Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

// CreateFolderIfNotExist creates a folder (directory marker) in MinIO if it doesn't exist
// In S3/MinIO, folders are virtual and created by adding a trailing slash to the key
func (m *MinioClient) CreateFolderIfNotExist(ctx context.Context, bucket, folderPath string) error {
	// Ensure bucket exists first
	if err := m.EnsureBucketByName(ctx, bucket); err != nil {
		return err
	}

	// Normalize folder path (ensure it ends with /)
	if !strings.HasSuffix(folderPath, "/") {
		folderPath = folderPath + "/"
	}

	// Check if folder marker already exists
	_, err := m.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(folderPath),
	})
	if err == nil {
		// Folder already exists
		return nil
	}

	// Create an empty object with trailing slash to represent the folder
	_, err = m.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(folderPath),
		Body:          bytes.NewReader([]byte{}),
		ContentLength: aws.Int64(0),
		ContentType:   aws.String("application/x-directory"),
	})
	if err != nil {
		return fmt.Errorf("failed to create folder marker: %w", err)
	}

	return nil
}
