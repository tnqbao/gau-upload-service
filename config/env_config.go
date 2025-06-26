package config 

type EnvConfig struct {
	CloudflareR2 struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
	} 
}

func NewConfig() *EnvConfig{
	CloudflareR2 {
		Endpoint := 
	}
}