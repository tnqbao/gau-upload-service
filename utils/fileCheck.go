package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

func CheckFileType(contentType string, allowedTypes []string) bool {
	for _, t := range allowedTypes {
		if t == contentType {
			return true
		}
	}
	return false
}

func IsFileSizeAllowed(size int64, maxSizeMB int64) bool {
	return size <= maxSizeMB*1024*1024
}

// SanitizeFileName cleans the filename by removing special characters and spaces
// Returns a clean filename in format: filename.ext
func SanitizeFileName(filename string) string {
	if filename == "" {
		return "file"
	}

	// Get file extension
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Remove or replace special characters and spaces
	// Keep only alphanumeric characters, hyphens, and underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	cleanName := reg.ReplaceAllString(nameWithoutExt, "_")

	// Remove multiple consecutive underscores
	reg2 := regexp.MustCompile(`_+`)
	cleanName = reg2.ReplaceAllString(cleanName, "_")

	// Remove leading and trailing underscores
	cleanName = strings.Trim(cleanName, "_")

	// If name becomes empty after cleaning, use default
	if cleanName == "" {
		cleanName = "file"
	}

	// Ensure extension starts with dot
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Clean extension as well
	if ext != "" {
		cleanExt := reg.ReplaceAllString(ext[1:], "")
		if cleanExt != "" {
			ext = "." + cleanExt
		} else {
			ext = ""
		}
	}

	return cleanName + ext
}
