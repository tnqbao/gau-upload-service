package main

import (
	"github.com/joho/godotenv"
	"github.com/tnqbao/gau-upload-service/config"
	"github.com/tnqbao/gau-upload-service/controller"
	"github.com/tnqbao/gau-upload-service/infra"
	"github.com/tnqbao/gau-upload-service/routes"
	"log"
)

func main() {
	err := godotenv.Load("/gau_upload/upload.env")
	if err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// Initialize configuration and infrastructure
	newConfig := config.NewConfig()
	newInfra := infra.InitInfra(newConfig)

	// Initialize controller with the new configuration and infrastructure
	ctrl := controller.NewController(newConfig, newInfra)

	router := routes.SetupRouter(ctrl)
	router.Run(":8082")
}