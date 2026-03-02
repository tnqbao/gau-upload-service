package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-upload-service/shared/utils"
)

func (ctrl *Controller) CheckHealth(c *gin.Context) {
	utils.JSON200(c, gin.H{"status": "running"})
}
