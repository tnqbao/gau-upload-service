package infra

import "github.com/tnqbao/gau-upload-service/config"

type Infra struct {
	CloudflareR2Client *CloudflareR2Client
	//PostgresClient     *Postgres
	//RedisClient        *Redis
}

func NewInfra(config *config.Config) *Infra {

	cloudflareR2Client, err := NewCloudflareR2Client(config.EnvConfig)
	if err != nil {
		panic("Failed to create Cloudflare R2 client: " + err.Error())
	}

	//postgresClient, err := NewPostgresClient(config.EnvConfig)
	//if err != nil {
	//	panic("Failed to create Postgres client: " + err.Error())
	//}

	return &Infra{
		CloudflareR2Client: cloudflareR2Client,
	}
}
