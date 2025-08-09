package controller

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/utils"
	"io"
	"net/http"
	"strings"
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
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.JSON400(c, "Failed to get file: "+err.Error())
		return
	}

	maxUploadSizeMB := ctrl.Config.EnvConfig.Limit.ImageMaxSize

	// Get file_path from form data to use as folder name
	folderName := strings.TrimSpace(c.PostForm("file_path"))
	if folderName == "" {
		utils.JSON400(c, "file_path is required")
		return
	}

	// Clean folder name: remove leading/trailing slashes and normalize
	folderName = strings.Trim(folderName, "/")
	if folderName == "" {
		utils.JSON400(c, "file_path cannot be empty or just slashes")
		return
	}

	// Sanitize the original filename to remove special characters and spaces
	sanitizedFileName := utils.SanitizeFileName(fileHeader.Filename)

	if !utils.IsFileSizeAllowed(fileHeader.Size, maxUploadSizeMB) {
		utils.JSON400(c, fmt.Sprintf("File size exceeds %dMB limit", maxUploadSizeMB))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		utils.JSON500(c, "Failed to open file: "+err.Error())
		return
	}
	defer file.Close()

	limited := io.LimitReader(file, maxUploadSizeMB)
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, limited)
	if err != nil {
		utils.JSON500(c, "Failed to read file: "+err.Error())
		return
	}
	if !utils.IsFileSizeAllowed(n, maxUploadSizeMB) {
		utils.JSON400(c, fmt.Sprintf("File size exceeds %dMB", maxUploadSizeMB))
		return
	}

	data := buf.Bytes()
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if !utils.CheckFileType(contentType, allowedContentTypes) {
		utils.JSON400(c, "Unsupported file type: "+contentType)
		return
	}

	// Construct the full path using sanitized filename: folder_name/sanitized_filename.ext
	fullPath := fmt.Sprintf("%s/%s", folderName, sanitizedFileName)

	if err := ctrl.Infrastructure.CloudflareR2Client.PutObject(c.Request.Context(), fullPath, data, contentType); err != nil {
		utils.JSON500(c, "Failed to upload file: "+err.Error())
		return
	}

	for i := range data {
		data[i] = 0
	}
	buf.Reset()

	filePath := fmt.Sprintf("%s", fullPath)
	utils.JSON200(c, gin.H{
		"file_path": filePath,
		"message":   "File uploaded successfully",
	})
}
