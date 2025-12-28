package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/tnqbao/gau-upload-service/shared/config"
)

// TempMinioClient is a separate MinIO client for temporary file storage
type TempMinioClient struct {
	Client *s3.Client
}

// NewTempMinioClient creates a new MinIO client for temp storage
func NewTempMinioClient(cfg *appconfig.EnvConfig) (*TempMinioClient, error) {
	endpoint := cfg.TempMinio.Endpoint
	accessKey := cfg.TempMinio.AccessKey
	secretKey := cfg.TempMinio.SecretKey
	region := cfg.TempMinio.Region
	useSSL := cfg.TempMinio.UseSSL

	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("temp MinIO configuration is incomplete")
	}

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
		return nil, fmt.Errorf("failed to load AWS config for Temp MinIO: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // MinIO requires path-style addressing
		if !useSSL {
			o.EndpointOptions.DisableHTTPS = true
		}
	})

	return &TempMinioClient{
		Client: s3Client,
	}, nil
}

// GetObjectStream streams an object from temp MinIO without loading into memory
func (m *TempMinioClient) GetObjectStream(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	resp, err := m.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object stream: %w", err)
	}

	contentLength := int64(0)
	if resp.ContentLength != nil {
		contentLength = *resp.ContentLength
	}

	return resp.Body, contentLength, nil
}

// GetObject retrieves an object from temp MinIO
func (m *TempMinioClient) GetObject(ctx context.Context, bucket, key string) ([]byte, string, error) {
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

// DeleteObject deletes an object from temp MinIO
func (m *TempMinioClient) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := m.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// HeadObject checks if an object exists and gets its metadata
func (m *TempMinioClient) HeadObject(ctx context.Context, bucket, key string) (*s3.HeadObjectOutput, error) {
	resp, err := m.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}
	return resp, nil
}
