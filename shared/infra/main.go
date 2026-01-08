package infra

import (
	"github.com/tnqbao/gau-upload-service/shared/config"
)

type Infra struct {
	MinioClient    *MinioClient
	ParquetService *ParquetService
	Logger         *LoggerClient
	RabbitMQ       *RabbitMQClient
}

func InitInfra(config *config.Config) *Infra {

	minioClient, err := NewMinioClient(config.EnvConfig)
	if err != nil {
		panic("Failed to create MinIO client: " + err.Error())
	}

	parquetService := NewParquetService(minioClient)

	loggerClient := InitLoggerClient(config.EnvConfig)
	if loggerClient == nil {
		panic("Failed to create Logger client")
	}

	// RabbitMQ is optional for HTTP service
	rabbitMQ := InitRabbitMQClient(config.EnvConfig)
	// Don't panic if RabbitMQ is not available - it's only needed for consumer

	return &Infra{
		MinioClient:    minioClient,
		ParquetService: parquetService,
		Logger:         loggerClient,
		RabbitMQ:       rabbitMQ,
	}
}

// InitInfraForConsumer initializes infrastructure specifically for the consumer service
// This requires RabbitMQ to be available
func InitInfraForConsumer(config *config.Config) *Infra {

	minioClient, err := NewMinioClient(config.EnvConfig)
	if err != nil {
		panic("Failed to create MinIO client: " + err.Error())
	}

	parquetService := NewParquetService(minioClient)

	loggerClient := InitLoggerClient(config.EnvConfig)
	if loggerClient == nil {
		panic("Failed to create Logger client")
	}

	rabbitMQ := InitRabbitMQClient(config.EnvConfig)
	if rabbitMQ == nil {
		panic("Failed to initialize RabbitMQ - required for consumer service")
	}

	return &Infra{
		MinioClient:    minioClient,
		ParquetService: parquetService,
		Logger:         loggerClient,
		RabbitMQ:       rabbitMQ,
	}
}
