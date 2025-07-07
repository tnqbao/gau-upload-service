package controller

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ctrl *Controller) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	key := c.PostForm("file_path")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_path is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot open uploaded file"})
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read uploaded file"})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	data := buf.Bytes()

	err = ctrl.Infrastructure.CloudflareR2Client.PutObject(c.Request.Context(), key, data, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload to R2"})
		return
	}

	// Reset the buffer to clear the data after upload
	for i := range data {
		data[i] = 0
	}
	buf.Reset()

	c.JSON(http.StatusOK, gin.H{
		"message":  "File uploaded successfully",
		"file_key": key,
	})
}
