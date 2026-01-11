package controller

import (
	"github.com/tnqbao/gau-upload-service/shared/config"
	"github.com/tnqbao/gau-upload-service/shared/infra"
	"github.com/tnqbao/gau-upload-service/shared/provider"
	"github.com/tnqbao/gau-upload-service/shared/repository"
)

type Controller struct {
	Repository     *repository.Repository
	Infrastructure *infra.Infra
	Config         *config.Config
	Provider       *provider.Provider
}

func NewController(cfg *config.Config, repo *repository.Repository, infra *infra.Infra) *Controller {
	provide := provider.InitProvider(cfg.EnvConfig)
	return &Controller{
		Repository:     repo,
		Infrastructure: infra,
		Config:         cfg,
		Provider:       provide,
	}
}
