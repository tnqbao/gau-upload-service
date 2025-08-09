package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/controller"
	"github.com/tnqbao/gau-upload-service/middlewares"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()
	middles, err := middlewares.NewMiddlewares(ctrl)
	if err != nil {
		panic(err)
	}

	apiRoutes := r.Group("/api/v2/upload")
	{
		apiRoutes.Use(middles.PrivateMiddlewares)
		apiRoutes.PATCH("/image", ctrl.UploadImage)

	}
	return r
}
