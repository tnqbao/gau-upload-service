package infra

import "github.com/tnqbao/gau-upload-service/config"

type Infra struct {
	MinioClient    *MinioClient
	ParquetService *ParquetService
	Logger         *LoggerClient
	//PostgresClient     *Postgres
	//RedisClient        *Redis
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

	//postgresClient, err := NewPostgresClient(config.EnvConfig)
	//if err != nil {
	//	panic("Failed to create Postgres client: " + err.Error())
	//}

	return &Infra{
		MinioClient:    minioClient,
		ParquetService: parquetService,
		Logger:         loggerClient,
	}
}
