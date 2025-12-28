package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/http/controller"
)

type Middlewares struct {
	PrivateMiddlewares gin.HandlerFunc
}

func NewMiddlewares(ctrl *controller.Controller) (*Middlewares, error) {
	private := PrivateMiddleware(ctrl.Config.EnvConfig)
	if private == nil {
		return nil, nil
	}

	return &Middlewares{
		PrivateMiddlewares: private,
	}, nil
}
