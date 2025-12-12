package config

import (
	"os"
	"strconv"
	"strings"
)

type EnvConfig struct {
	Minio struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		Region    string
		UseSSL    bool
	}

	PrivateKey string

	Limit struct {
		ImageMaxSize int64
		FileMaxSize  int64
	}

	Grafana struct {
		OTLPEndpoint string
		ServiceName  string
	}

	Environment struct {
		Mode  string
		Group string
	}
}

func LoadEnvConfig() *EnvConfig {
	var config EnvConfig

	// Minio
	config.Minio.Endpoint = os.Getenv("MINIO_ENDPOINT")
	config.Minio.AccessKey = os.Getenv("MINIO_ACCESS_KEY_ID")
	config.Minio.SecretKey = os.Getenv("MINIO_SECRET_ACCESS_KEY")
	config.Minio.Region = os.Getenv("MINIO_REGION")
	if config.Minio.Region == "" {
		config.Minio.Region = "us-east-1"
	}
	useSSL := os.Getenv("MINIO_USE_SSL")
	config.Minio.UseSSL = useSSL == "true" || useSSL == "1"

	config.PrivateKey = os.Getenv("PRIVATE_KEY")

	if imageSizeStr := os.Getenv("IMAGE_MAX_SIZE"); imageSizeStr != "" {
		if imageSize, err := strconv.ParseInt(imageSizeStr, 10, 64); err == nil {
			config.Limit.ImageMaxSize = imageSize
		} else {
			config.Limit.ImageMaxSize = 5242880 // Default to 5MB in bytes if invalid
		}
	} else {
		config.Limit.ImageMaxSize = 5242880 // Default to 5MB in bytes if not set
	}

	if fileSizeStr := os.Getenv("FILE_MAX_SIZE"); fileSizeStr != "" {
		if fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64); err == nil {
			config.Limit.FileMaxSize = fileSize
		} else {
			config.Limit.FileMaxSize = 10485760 // Default to 10MB in bytes if invalid
		}
	} else {
		config.Limit.FileMaxSize = 10485760 // Default to 10MB in bytes if not set
	}

	// Grafana/OpenTelemetry
	grafanaEndpoint := os.Getenv("GRAFANA_OTLP_ENDPOINT")
	if grafanaEndpoint == "" {
		grafanaEndpoint = "https://grafana.gauas.online"
	}
	// Remove protocol for OpenTelemetry client to avoid duplicate protocols
	if strings.HasPrefix(grafanaEndpoint, "https://") {
		config.Grafana.OTLPEndpoint = strings.TrimPrefix(grafanaEndpoint, "https://")
	} else if strings.HasPrefix(grafanaEndpoint, "http://") {
		config.Grafana.OTLPEndpoint = strings.TrimPrefix(grafanaEndpoint, "http://")
	} else {
		config.Grafana.OTLPEndpoint = grafanaEndpoint
	}
	config.Grafana.ServiceName = os.Getenv("SERVICE_NAME")
	if config.Grafana.ServiceName == "" {
		config.Grafana.ServiceName = "gau-account-service"
	}

	config.Environment.Mode = os.Getenv("DEPLOY_ENV")
	if config.Environment.Mode == "" {
		config.Environment.Mode = "development"
	}

	config.Environment.Group = os.Getenv("GROUP_NAME")
	if config.Environment.Group == "" {
		config.Environment.Group = "local"
	}

	return &config
}
