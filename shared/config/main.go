package config

type Config struct {
	EnvConfig *EnvConfig
}

func NewConfig() *Config {
	EnvConfig := LoadEnvConfig()
	return &Config{
		EnvConfig: EnvConfig,
	}
}