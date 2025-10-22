package config

import (
	"os"
	"strconv"
	"strings"
)

type EnvConfig struct {
	CloudflareR2 struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
	}

	PrivateKey string

	Limit struct {
		ImageMaxSize int64
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

	// Cloudflare R2
	config.CloudflareR2.Endpoint = os.Getenv("CLOUDFLARE_R2_ENDPOINT")
	config.CloudflareR2.AccessKey = os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	config.CloudflareR2.SecretKey = os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	if bucketName := os.Getenv("CLOUDFLARE_R2_BUCKET_NAME"); bucketName != "" {
		config.CloudflareR2.BucketName = bucketName
	} else {
		config.CloudflareR2.BucketName = "default-bucket"
	}

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
