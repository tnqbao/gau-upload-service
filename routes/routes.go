package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/controller"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()
	apiRoutes := r.Group("/api/v2/upload")
	{
		apiRoutes.PATCH("/image", ctrl.UploadImage)

	}
	return r
}
