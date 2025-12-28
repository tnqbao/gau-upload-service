package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tnqbao/gau-upload-service/shared/config"
	"github.com/tnqbao/gau-upload-service/shared/infra"
)

// ChunkRequest represents a request to process a large file
type ChunkRequest struct {
	TempBucket   string
	TempPath     string
	TargetBucket string
	TargetFolder string
	OriginalName string
	FileHash     string
	FileSize     int64
	ChunkSize    int64
	Metadata     map[string]string
}

// ProcessResult represents the result of processing a large file
type ProcessResult struct {
	OriginalFile string            `json:"original_file"`
	FileHash     string            `json:"file_hash"`
	TotalSize    int64             `json:"total_size"`
	FilePath     string            `json:"file_path"`
	ContentType  string            `json:"content_type"`
	CreatedAt    time.Time         `json:"created_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ChunkerService handles large file processing operations
type ChunkerService struct {
	config           *config.Config
	infra            *infra.Infra
	defaultChunkSize int64
	tempDir          string
}

// NewChunkerService creates a new chunker service
func NewChunkerService(cfg *config.Config, inf *infra.Infra) *ChunkerService {
	return &ChunkerService{
		config:           cfg,
		infra:            inf,
		defaultChunkSize: cfg.EnvConfig.ChunkConfig.DefaultChunkSize,
		tempDir:          cfg.EnvConfig.ChunkConfig.TempDir,
	}
}

// ProcessFile downloads a file from temp MinIO and uploads to main MinIO as a single file
// The file is downloaded in chunks to handle large files, but saved as a single file at destination
func (s *ChunkerService) ProcessFile(ctx context.Context, req ChunkRequest) (*ProcessResult, error) {
	log.Printf("[Chunker] Processing file from %s/%s (size: %d bytes)",
		req.TempBucket, req.TempPath, req.FileSize)

	// Ensure temp directory exists
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Download file from temp MinIO to local disk
	localPath, err := s.downloadFromTemp(ctx, req.TempBucket, req.TempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download from temp: %w", err)
	}
	defer os.Remove(localPath) // Clean up local file

	// Get file info
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}

	// Determine content type from metadata or default
	contentType := "application/octet-stream"
	if ct, ok := req.Metadata["content_type"]; ok && ct != "" {
		contentType = ct
	}

	// Construct final file path: customPath/hash.ext
	ext := filepath.Ext(req.OriginalName)
	if ext == "" {
		ext = filepath.Ext(req.TempPath)
	}

	var finalPath string
	if req.TargetFolder != "" && req.TargetFolder != req.FileHash {
		// If TargetFolder contains customPath/hash, extract just the customPath
		customPath := req.Metadata["custom_path"]
		if customPath != "" {
			finalPath = fmt.Sprintf("%s/%s%s", customPath, req.FileHash, ext)
		} else {
			finalPath = fmt.Sprintf("%s%s", req.FileHash, ext)
		}
	} else {
		finalPath = fmt.Sprintf("%s%s", req.FileHash, ext)
	}

	log.Printf("[Chunker] Uploading file to %s/%s", req.TargetBucket, finalPath)

	// Upload file to main MinIO
	if err := s.uploadToMain(ctx, localPath, req.TargetBucket, finalPath, contentType, req); err != nil {
		return nil, fmt.Errorf("failed to upload to main MinIO: %w", err)
	}

	// Clean up temp file in temp MinIO
	if err := s.cleanupTemp(ctx, req.TempBucket, req.TempPath); err != nil {
		log.Printf("[Chunker] Warning: failed to cleanup temp file: %v", err)
		// Don't fail the whole operation for cleanup failure
	}

	result := &ProcessResult{
		OriginalFile: req.OriginalName,
		FileHash:     req.FileHash,
		TotalSize:    fileInfo.Size(),
		FilePath:     finalPath,
		ContentType:  contentType,
		CreatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	log.Printf("[Chunker] Successfully processed file: %s -> %s/%s",
		req.OriginalName, req.TargetBucket, finalPath)

	return result, nil
}

// downloadFromTemp downloads a file from temp MinIO to local disk
func (s *ChunkerService) downloadFromTemp(ctx context.Context, bucket, path string) (string, error) {
	log.Printf("[Chunker] Downloading file from temp MinIO: %s/%s", bucket, path)

	// Get object stream
	stream, _, err := s.infra.TempMinioClient.GetObjectStream(ctx, bucket, path)
	if err != nil {
		return "", fmt.Errorf("failed to get object stream: %w", err)
	}
	defer stream.Close()

	// Create local temp file
	localPath := filepath.Join(s.tempDir, filepath.Base(path))
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy stream to local file
	written, err := io.Copy(file, stream)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("failed to write to local file: %w", err)
	}

	log.Printf("[Chunker] Downloaded %d bytes to %s", written, localPath)
	return localPath, nil
}

// uploadToMain uploads a local file to main MinIO
func (s *ChunkerService) uploadToMain(ctx context.Context, localPath, bucket, key, contentType string, req ChunkRequest) error {
	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file into memory (for files under 5GB this is acceptable)
	// For very large files, we could implement multipart upload
	data := make([]byte, info.Size())
	_, err = io.ReadFull(file, data)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Prepare metadata
	metadata := map[string]string{
		"file-hash":     req.FileHash,
		"original-name": req.OriginalName,
		"content-type":  contentType,
	}
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	// Upload to main MinIO
	if err := s.infra.MinioClient.PutObjectWithMetadata(ctx, bucket, key, data, contentType, metadata); err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	log.Printf("[Chunker] Uploaded %d bytes to %s/%s", info.Size(), bucket, key)
	return nil
}

// cleanupTemp deletes the temporary file from temp MinIO
func (s *ChunkerService) cleanupTemp(ctx context.Context, bucket, path string) error {
	if err := s.infra.TempMinioClient.DeleteObject(ctx, bucket, path); err != nil {
		return fmt.Errorf("failed to delete temp file: %w", err)
	}
	log.Printf("[Chunker] Cleaned up temp file: %s/%s", bucket, path)
	return nil
}
