package controller

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/infra"
	"github.com/tnqbao/gau-upload-service/utils"
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

	maxUploadSize := ctrl.Config.EnvConfig.Limit.FileMaxSize

	if fileHeader.Size > maxUploadSize {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload File] File size exceeds limit: %d bytes", fileHeader.Size)
		utils.JSON400(c, fmt.Sprintf("File size exceeds %d bytes limit", maxUploadSize))
		return
	}

	// Open and read file
	file, err := fileHeader.Open()
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to open uploaded file")
		utils.JSON500(c, "Failed to open file: "+err.Error())
		return
	}
	defer file.Close()

	// Read file data
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to read uploaded file")
		utils.JSON500(c, "Failed to read file: "+err.Error())
		return
	}

	data := buf.Bytes()

	// Calculate SHA-256 hash for deduplication
	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Detect content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Get file extension
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = getExtensionFromContentType(contentType)
	}

	// Construct file path with proper directory structure
	// MinIO automatically creates folders when uploading with path separators
	var fullPath string
	if customPath != "" {
		// Path will be: path/hash.ext (e.g., "abc/def/abc123.jpg")
		fullPath = fmt.Sprintf("%s/%s%s", customPath, fileHash, ext)
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Upload to path: %s", fullPath)
	} else {
		// No path specified, save to root: hash.ext
		fullPath = fmt.Sprintf("%s%s", fileHash, ext)
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Upload to root: %s", fullPath)
	}

	// If custom path provided, ensure folders exist in MinIO FIRST (before checking deduplication)
	if customPath != "" {
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Creating folder structure for path: %s", customPath)
		segments := strings.Split(customPath, "/")
		for i := 0; i < len(segments); i++ {
			folder := strings.Join(segments[:i+1], "/")
			ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Creating folder: %s", folder)
			if err := ctrl.Infrastructure.MinioClient.CreateFolderIfNotExist(ctx, bucketName, folder); err != nil {
				ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload File] Failed to create folder in MinIO: %s", folder)
				utils.JSON500(c, "Failed to create folder: "+err.Error())
				return
			}
			ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload File] Successfully created folder: %s/", folder)
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

	// Upload file with metadata to MinIO
	metadata := map[string]string{
		"file-hash":     fileHash,
		"original-name": fileHeader.Filename,
		"content-type":  contentType,
	}

	if err := ctrl.Infrastructure.MinioClient.PutObjectWithMetadata(ctx, bucketName, fullPath, data, contentType, metadata); err != nil {
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

	// Clear sensitive data from memory
	for i := range data {
		data[i] = 0
	}
	buf.Reset()

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
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Get File] Failed to get file from MinIO")
		utils.JSON404(c, "File not found: "+err.Error())
		return
	}

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
