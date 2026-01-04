package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/shared/infra"
	"github.com/tnqbao/gau-upload-service/shared/utils"
)

// UploadFile handles generic file upload with deduplication using Parquet
func (ctrl *Controller) UploadFile(c *gin.Context) {
	ctx := c.Request.Context()
	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Upload request received")

	// Get file from multipart form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to get file from form data")
		utils.JSON400(c, "Failed to get file: "+err.Error())
		return
	}

	// Get bucket name from form data
	bucketName := strings.TrimSpace(c.PostForm("bucket"))
	if bucketName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload File] bucket is required")
		utils.JSON400(c, "bucket parameter is required")
		return
	}

	// Optional: Get custom file path/folder (supports nested paths like abc/def)
	customPath := strings.TrimSpace(c.PostForm("path"))
	if customPath != "" {
		// Clean and normalize path: remove leading/trailing slashes, replace backslashes
		customPath = strings.Trim(customPath, "/\\")
		customPath = strings.ReplaceAll(customPath, "\\", "/")

		// Remove any double slashes
		for strings.Contains(customPath, "//") {
			customPath = strings.ReplaceAll(customPath, "//", "/")
		}

		// Validate path doesn't contain dangerous characters
		if strings.Contains(customPath, "..") {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload File] Invalid path contains ..")
			utils.JSON400(c, "Invalid path: path cannot contain '..'")
			return
		}
	}

	// Optional: Get is_hash parameter (defaults to true for backward compatibility)
	isHashStr := strings.ToLower(strings.TrimSpace(c.PostForm("is_hash")))
	isHash := true // default is true
	if isHashStr == "false" || isHashStr == "0" {
		isHash = false
	}

	maxUploadSize := ctrl.Config.EnvConfig.Limit.FileMaxSize

	if fileHeader.Size > maxUploadSize {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload File] File size exceeds limit: %d bytes", fileHeader.Size)
		utils.JSON400(c, fmt.Sprintf("File size exceeds %d bytes limit", maxUploadSize))
		return
	}

	// Open uploaded file stream
	srcFile, err := fileHeader.Open()
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to open uploaded file")
		utils.JSON500(c, "Failed to open file: "+err.Error())
		return
	}
	defer srcFile.Close()

	// Create a temporary file to store the content while hashing
	tempDir := ctrl.Config.EnvConfig.ChunkConfig.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	// Ensure temp dir exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to create temp dir")
		utils.JSON500(c, "Failed to create temp dir: "+err.Error())
		return
	}

	tempFile, err := os.CreateTemp(tempDir, "upload-*")
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to create temp file")
		utils.JSON500(c, "Failed to create temp file: "+err.Error())
		return
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name()) // Clean up temp file
	}()

	// Create hasher
	hasher := sha256.New()

	// Stream from source to both temp file and hasher
	writer := io.MultiWriter(tempFile, hasher)

	// Copy data
	if _, err := io.Copy(writer, srcFile); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to stream file to temp storage")
		utils.JSON500(c, "Failed to stream file: "+err.Error())
		return
	}

	// Calculate SHA-256 hash
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Detect content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// If content type is missing, we need to detect it.
		// We can read the first 512 bytes from the temp file.
		if _, err := tempFile.Seek(0, 0); err != nil {
			ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to seek temp file")
			utils.JSON500(c, "Failed to seek file: "+err.Error())
			return
		}
		buffer := make([]byte, 512)
		n, _ := tempFile.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
	}

	// Get file extension
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = getExtensionFromContentType(contentType)
	}

	// Construct file name based on is_hash parameter
	var fileName string
	if isHash {
		// Use hash as filename
		fileName = fileHash + ext
	} else {
		// Use original filename, but sanitize it
		fileName = utils.SanitizeFileName(fileHeader.Filename)
		// Ensure the sanitized name has the correct extension
		if filepath.Ext(fileName) == "" && ext != "" {
			fileName = fileName + ext
		}
	}

	// Construct file path with proper directory structure
	var fullPath string
	if customPath != "" {
		// Path will be: path/filename (e.g., "abc/def/file.jpg" or "abc/def/abc123.jpg")
		fullPath = fmt.Sprintf("%s/%s", customPath, fileName)
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Upload to path: %s", fullPath)
	} else {
		// No path specified, save to root: filename
		fullPath = fileName
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Upload to root: %s", fullPath)
	}

	// If custom path provided, ensure folders exist in MinIO FIRST
	if customPath != "" {
		// ... existing folder creation logic ...
		segments := strings.Split(customPath, "/")
		for i := 0; i < len(segments); i++ {
			folder := strings.Join(segments[:i+1], "/")
			if err := ctrl.Infrastructure.MinioClient.CreateFolderIfNotExist(ctx, bucketName, folder); err != nil {
				// Don't fail hard on folder creation if it might exist
				ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload File] Warning: Failed to create folder %s: %v", folder, err)
			}
		}
	}

	// Check if file already exists by hash in metadata using Parquet
	existingFile, exists, err := ctrl.Infrastructure.ParquetService.CheckFileByHash(ctx, bucketName, fileHash)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to check file existence")
		utils.JSON500(c, "Failed to check file existence: "+err.Error())
		return
	}

	if exists {
		// File with same hash exists, but check if it's at a different path
		if existingFile == fullPath {
			// Same file at same path - true duplicate
			ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] File already exists at exact path: %s (hash: %s)", existingFile, fileHash)
			utils.JSON200(c, gin.H{
				"file_path":    existingFile,
				"file_hash":    fileHash,
				"message":      "File already exists (deduplicated)",
				"bucket":       bucketName,
				"content_type": contentType,
				"size":         fileHeader.Size,
				"duplicated":   true,
			})
			return
		} else {
			// Same content but different path - create a copy/reference
			ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] File with same hash exists at %s, but creating copy at new path: %s", existingFile, fullPath)
			// Continue to upload the file at the new path
		}
	}

	// Prepare to upload from temp file
	// Reset temp file pointer to beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to seek temp file for upload")
		utils.JSON500(c, "Failed to prepare file for upload: "+err.Error())
		return
	}

	// Upload file stream with metadata to MinIO
	metadata := map[string]string{
		"file-hash":     fileHash,
		"original-name": fileHeader.Filename,
		"content-type":  contentType,
	}

	// Use Stream upload
	if err := ctrl.Infrastructure.MinioClient.PutObjectStreamWithMetadata(ctx, bucketName, fullPath, tempFile, fileHeader.Size, contentType, metadata); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to upload file to MinIO")
		utils.JSON500(c, "Failed to upload file: "+err.Error())
		return
	}

	// Add metadata to Parquet for fast lookup
	fileMetadata := infra.FileMetadata{
		FileHash:     fileHash,
		FilePath:     fullPath,
		BucketName:   bucketName,
		OriginalName: fileHeader.Filename,
		ContentType:  contentType,
		FileSize:     fileHeader.Size,
		UploadedAt:   time.Now(),
	}
	if err := ctrl.Infrastructure.ParquetService.AddFileMetadata(ctx, fileMetadata); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to save metadata to Parquet")
		// Don't fail the request, just log the error
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] File uploaded successfully: %s (hash: %s)", fullPath, fileHash)
	utils.JSON200(c, gin.H{
		"file_path":    fullPath,
		"file_hash":    fileHash,
		"message":      "File uploaded successfully",
		"bucket":       bucketName,
		"content_type": contentType,
		"size":         fileHeader.Size,
		"duplicated":   exists && existingFile != fullPath, // True if file content was duplicated but saved at new path
	})
}

// GetFile retrieves a file from MinIO
func (ctrl *Controller) GetFile(c *gin.Context) {
	ctx := c.Request.Context()
	filePath := c.Query("file_path")
	bucketName := c.Query("bucket")

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Get File] Request received - Bucket: %s, Path: %s", bucketName, filePath)

	if filePath == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Get File] file_path is required")
		utils.JSON400(c, "file_path is required")
		return
	}

	if bucketName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Get File] bucket is required")
		utils.JSON400(c, "bucket parameter is required")
		return
	}

	data, contentType, err := ctrl.Infrastructure.MinioClient.GetObjectFromBucket(ctx, bucketName, filePath)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Get File] Failed to get file from MinIO - Bucket: %s, Path: %s, Error: %v", bucketName, filePath, err)
		utils.JSON404(c, "File not found: "+err.Error())
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Get File] File retrieved successfully - Bucket: %s, Path: %s, ContentType: %s, Size: %d bytes", bucketName, filePath, contentType, len(data))
	c.Data(http.StatusOK, contentType, data)
}

// DeleteFile deletes a file from MinIO and removes metadata from Parquet
func (ctrl *Controller) DeleteFile(c *gin.Context) {
	ctx := c.Request.Context()
	filePath := c.Query("file_path")
	bucketName := c.Query("bucket")

	if filePath == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Delete File] file_path is required")
		utils.JSON400(c, "file_path is required")
		return
	}

	if bucketName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Delete File] bucket is required")
		utils.JSON400(c, "bucket parameter is required")
		return
	}

	// Delete file from MinIO
	if err := ctrl.Infrastructure.MinioClient.DeleteObjectFromBucket(ctx, bucketName, filePath); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Delete File] Failed to delete file from MinIO")
		utils.JSON500(c, "Failed to delete file: "+err.Error())
		return
	}

	// Remove metadata from Parquet
	if err := ctrl.Infrastructure.ParquetService.RemoveFileMetadata(ctx, bucketName, filePath); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Delete File] Failed to remove metadata from Parquet")
		// Don't fail the request, just log the error
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Delete File] File deleted successfully: %s", filePath)
	utils.JSON200(c, gin.H{
		"file_path": filePath,
		"bucket":    bucketName,
		"message":   "File deleted successfully",
	})
}

// ListFiles lists all files in a bucket with optional prefix
func (ctrl *Controller) ListFiles(c *gin.Context) {
	ctx := c.Request.Context()
	prefix := c.Query("prefix")
	bucketName := c.Query("bucket")

	if bucketName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[List Files] bucket is required")
		utils.JSON400(c, "bucket parameter is required")
		return
	}

	files, err := ctrl.Infrastructure.MinioClient.ListObjectsFromBucket(ctx, bucketName, prefix)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[List Files] Failed to list files from MinIO")
		utils.JSON500(c, "Failed to list files: "+err.Error())
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[List Files] Listed %d files with prefix: %s in bucket: %s", len(files), prefix, bucketName)
	utils.JSON200(c, gin.H{
		"files":  files,
		"count":  len(files),
		"bucket": bucketName,
		"prefix": prefix,
	})
}

// Helper function to get extension from content type
func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/json":
		return ".json"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg":
		return ".mp3"
	default:
		return ".bin"
	}
}
