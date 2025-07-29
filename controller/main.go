package controller

import (
	"github.com/tnqbao/gau-upload-service/config"
	"github.com/tnqbao/gau-upload-service/infra"
	"github.com/tnqbao/gau-upload-service/repository"
)

type Controller struct {
	Repository     *repository.Repository
	Infrastructure *infra.Infra
	Config         *config.Config
}

func NewController(cfg *config.Config, repo *repository.Repository, infra *infra.Infra) *Controller {
	return &Controller{
		Repository:     repo,
		Infrastructure: infra,
		Config:         cfg,
	}
}
