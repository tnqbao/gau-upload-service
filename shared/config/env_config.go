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

	TempMinio struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		Region    string
		UseSSL    bool
	}

	RabbitMQ struct {
		Host     string
		Port     string
		Username string
		Password string
	}

	ChunkConfig struct {
		DefaultChunkSize int64
		MaxChunkSize     int64
		TempDir          string
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

	// Main Minio
	config.Minio.Endpoint = os.Getenv("MINIO_ENDPOINT")
	config.Minio.AccessKey = os.Getenv("MINIO_ACCESS_KEY_ID")
	config.Minio.SecretKey = os.Getenv("MINIO_SECRET_ACCESS_KEY")
	config.Minio.Region = os.Getenv("MINIO_REGION")
	if config.Minio.Region == "" {
		config.Minio.Region = "us-east-1"
	}
	useSSL := os.Getenv("MINIO_USE_SSL")
	config.Minio.UseSSL = useSSL == "true" || useSSL == "1"

	// Temp Minio (for large file uploads)
	config.TempMinio.Endpoint = os.Getenv("TEMP_MINIO_ENDPOINT")
	if config.TempMinio.Endpoint == "" {
		config.TempMinio.Endpoint = config.Minio.Endpoint // Default to main MinIO
	}
	config.TempMinio.AccessKey = os.Getenv("TEMP_MINIO_ACCESS_KEY_ID")
	if config.TempMinio.AccessKey == "" {
		config.TempMinio.AccessKey = config.Minio.AccessKey
	}
	config.TempMinio.SecretKey = os.Getenv("TEMP_MINIO_SECRET_ACCESS_KEY")
	if config.TempMinio.SecretKey == "" {
		config.TempMinio.SecretKey = config.Minio.SecretKey
	}
	config.TempMinio.Region = os.Getenv("TEMP_MINIO_REGION")
	if config.TempMinio.Region == "" {
		config.TempMinio.Region = config.Minio.Region
	}
	tempUseSSL := os.Getenv("TEMP_MINIO_USE_SSL")
	if tempUseSSL != "" {
		config.TempMinio.UseSSL = tempUseSSL == "true" || tempUseSSL == "1"
	} else {
		config.TempMinio.UseSSL = config.Minio.UseSSL
	}

	// RabbitMQ
	config.RabbitMQ.Host = os.Getenv("RABBITMQ_HOST")
	if config.RabbitMQ.Host == "" {
		config.RabbitMQ.Host = "localhost"
	}
	config.RabbitMQ.Port = os.Getenv("RABBITMQ_PORT")
	if config.RabbitMQ.Port == "" {
		config.RabbitMQ.Port = "5672"
	}
	config.RabbitMQ.Username = os.Getenv("RABBITMQ_USER")
	if config.RabbitMQ.Username == "" {
		config.RabbitMQ.Username = "guest"
	}
	config.RabbitMQ.Password = os.Getenv("RABBITMQ_PASSWORD")
	if config.RabbitMQ.Password == "" {
		config.RabbitMQ.Password = "guest"
	}

	// Chunk Configuration
	if chunkSizeStr := os.Getenv("DEFAULT_CHUNK_SIZE"); chunkSizeStr != "" {
		if chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64); err == nil {
			config.ChunkConfig.DefaultChunkSize = chunkSize
		} else {
			config.ChunkConfig.DefaultChunkSize = 10485760 // Default to 10MB
		}
	} else {
		config.ChunkConfig.DefaultChunkSize = 10485760 // Default to 10MB
	}

	if maxChunkSizeStr := os.Getenv("MAX_CHUNK_SIZE"); maxChunkSizeStr != "" {
		if maxChunkSize, err := strconv.ParseInt(maxChunkSizeStr, 10, 64); err == nil {
			config.ChunkConfig.MaxChunkSize = maxChunkSize
		} else {
			config.ChunkConfig.MaxChunkSize = 104857600 // Default to 100MB
		}
	} else {
		config.ChunkConfig.MaxChunkSize = 104857600 // Default to 100MB
	}

	config.ChunkConfig.TempDir = os.Getenv("TEMP_DIR")
	if config.ChunkConfig.TempDir == "" {
		config.ChunkConfig.TempDir = "/tmp/gau-upload"
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
		config.Grafana.ServiceName = "gau-upload-service"
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
