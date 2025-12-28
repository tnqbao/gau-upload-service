package repository

import (
	"github.com/tnqbao/gau-upload-service/shared/config"
)

type Repository struct {
}

func NewRepository(config *config.Config) *Repository {
	return &Repository{}
}
