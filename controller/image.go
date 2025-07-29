package controller

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/utils"
	"io"
	"net/http"
)

const maxUploadSizeMB = 10

var allowedContentTypes = []string{"image/jpeg", "image/png", "image/webp"}

func (ctrl *Controller) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.JSON400(c, "Failed to get file: "+err.Error())
		return
	}

	key := c.PostForm("file_path")
	if key == "" {
		utils.JSON400(c, "file_path is required")
		return
	}

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

	limited := io.LimitReader(file, maxUploadSizeMB*1024*1024+1)
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

	if err := ctrl.Infrastructure.CloudflareR2Client.PutObject(c.Request.Context(), key, data, contentType); err != nil {
		utils.JSON500(c, "Failed to upload file: "+err.Error())
		return
	}

	for i := range data {
		data[i] = 0
	}
	buf.Reset()

	filePath := fmt.Sprintf("%s/%s", key, fileHeader.Filename)
	utils.JSON200(c, gin.H{
		"file_path": filePath,
		"message":   "File uploaded successfully",
	})
}
