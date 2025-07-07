package repository

import "github.com/tnqbao/gau-upload-service/config"

type Repository struct {
}

func NewRepository(config *config.Config) *Repository {
	return &Repository{}
}
