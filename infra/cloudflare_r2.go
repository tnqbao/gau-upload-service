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
	appconfig "github.com/tnqbao/gau-upload-service/config"
)

type CloudflareR2Client struct {
	Client     *s3.Client
	BucketName string
}

func NewCloudflareR2Client(cfg *appconfig.EnvConfig) (*CloudflareR2Client, error) {
	endpoint := cfg.CloudflareR2.Endpoint
	bucketName := cfg.CloudflareR2.BucketName
	accessKey := cfg.CloudflareR2.AccessKey
	secret := cfg.CloudflareR2.SecretKey

	// Create a custom endpoint resolver for Cloudflare R2 || because int not use default AWS endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           endpoint,
			SigningRegion: "auto",
		}, nil
	})

	// Load AWS configuration with custom endpoint and credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secret, "")),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	return &CloudflareR2Client{
		Client:     s3Client,
		BucketName: bucketName,
	}, nil
}

func (r2 *CloudflareR2Client) GetObject(ctx context.Context, key string) ([]byte, string, error) {
	resp, err := r2.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r2.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object: %w", err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, "", err
	}

	contentType := aws.ToString(resp.ContentType)
	return buf.Bytes(), contentType, nil
}

func (r2 *CloudflareR2Client) GetObjectWithLimit(ctx context.Context, key string, maxSize int64) ([]byte, string, error) {
	resp, err := r2.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r2.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxSize+1)
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, limited)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read object: %w", err)
	}
	if n > maxSize {
		return nil, "", fmt.Errorf("object too large (%d bytes)", n)
	}

	contentType := aws.ToString(resp.ContentType)
	return buf.Bytes(), contentType, nil
}

func (r2 *CloudflareR2Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := r2.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r2.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (r2 *CloudflareR2Client) DeleteObject(ctx context.Context, key string) error {
	_, err := r2.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r2.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (r2 *CloudflareR2Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	resp, err := r2.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(r2.BucketName),
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
