package controller

import (
	"bytes"
	"github.com/tnqbao/gau-upload-service/utils"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

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

	file, err := fileHeader.Open()
	if err != nil {
		utils.JSON500(c, "Failed to open file: "+err.Error())
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		utils.JSON500(c, "Failed to read file: "+err.Error())
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	data := buf.Bytes()

	err = ctrl.Infrastructure.CloudflareR2Client.PutObject(c.Request.Context(), key, data, contentType)
	if err != nil {
		utils.JSON500(c, "Failed to upload file: "+err.Error())
		return
	}
	// Reset the buffer to clear the data after upload
	for i := range data {
		data[i] = 0
	}
	buf.Reset()

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"url":     key,
	})
}
