package config

import (
	"os"
)

type EnvConfig struct {
	CloudflareR2 struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
	}

	PrivateKey string
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

	return &config
}
