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

func NewController() *Controller {
	cfg := config.NewConfig()
	repo := repository.NewRepository(cfg)
	infra := infra.NewInfra(cfg)

	return &Controller{
		Repository:     repo,
		Infrastructure: infra,
		Config:         cfg,
	}
}
