package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/utils"
)

var allowedContentTypes = []string{
	"image/jpeg",
	"image/jpg",
	"image/png",
	"image/webp",
	"image/svg+xml",
	"image/x-icon",
	"image/vnd.microsoft.icon",
}

func (ctrl *Controller) UploadImage(c *gin.Context) {
	ctx := c.Request.Context()
	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload Image] Create new token request received")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload Image] Failed to get file from form data")
		utils.JSON400(c, "Failed to get file: "+err.Error())
		return
	}

	maxUploadSizeMB := ctrl.Config.EnvConfig.Limit.ImageMaxSize

	// Get file_path from form data to use as folder name
	folderName := strings.TrimSpace(c.PostForm("file_path"))
	if folderName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload Image] file_path is required")
		utils.JSON400(c, "file_path is required")
		return
	}

	// Clean folder name: remove leading/trailing slashes and normalize
	folderName = strings.Trim(folderName, "/")
	if folderName == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload Image] file_path cannot be empty or just slashes")
		utils.JSON400(c, "file_path cannot be empty or just slashes")
		return
	}

	// Sanitize the original filename to remove special characters and spaces
	sanitizedFileName := utils.SanitizeFileName(fileHeader.Filename)

	if !utils.IsFileSizeAllowed(fileHeader.Size, maxUploadSizeMB) {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload Image] File size exceeds limit: %dMB", maxUploadSizeMB)
		utils.JSON400(c, fmt.Sprintf("File size exceeds %dMB limit", maxUploadSizeMB))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload Image] Failed to open uploaded file")
		utils.JSON500(c, "Failed to open file: "+err.Error())
		return
	}
	defer file.Close()

	limited := io.LimitReader(file, maxUploadSizeMB)
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, limited)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload Image] Failed to read uploaded file")
		utils.JSON500(c, "Failed to read file: "+err.Error())
		return
	}
	if !utils.IsFileSizeAllowed(n, maxUploadSizeMB) {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload Image] File size exceeds limit after reading: %dMB", maxUploadSizeMB)
		utils.JSON400(c, fmt.Sprintf("File size exceeds %dMB", maxUploadSizeMB))
		return
	}

	data := buf.Bytes()
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if !utils.CheckFileType(contentType, allowedContentTypes) {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[Upload Image] Unsupported file type: %s", contentType)
		utils.JSON400(c, "Unsupported file type: "+contentType)
		return
	}

	// Construct the full path using sanitized filename: folder_name/sanitized_filename.ext
	fullPath := fmt.Sprintf("%s/%s", folderName, sanitizedFileName)

	if err := ctrl.Infrastructure.CloudflareR2Client.PutObject(c.Request.Context(), fullPath, data, contentType); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[Upload Image] Failed to upload file to Cloudflare R2")
		utils.JSON500(c, "Failed to upload file: "+err.Error())
		return
	}

	for i := range data {
		data[i] = 0
	}
	buf.Reset()

	filePath := fmt.Sprintf("%s", fullPath)
	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[Upload Image] File uploaded successfully: %s", filePath)
	utils.JSON200(c, gin.H{
		"file_path": filePath,
		"message":   "File uploaded successfully",
	})
}
