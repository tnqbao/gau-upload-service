package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/http/controller"
	"github.com/tnqbao/gau-upload-service/http/middlewares"
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

		// Generic file upload endpoints
		apiRoutes.POST("/file", ctrl.UploadFile)
		apiRoutes.GET("/file", ctrl.GetFile)
		apiRoutes.DELETE("/file", ctrl.DeleteFile)
		apiRoutes.GET("/files/list", ctrl.ListFiles)
	}
	return r
}
