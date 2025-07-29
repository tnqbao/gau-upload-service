package utils

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
