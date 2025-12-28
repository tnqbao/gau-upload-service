package main

import (
	"github.com/joho/godotenv"
	"github.com/tnqbao/gau-upload-service/http/controller"
	"github.com/tnqbao/gau-upload-service/http/routes"
	"github.com/tnqbao/gau-upload-service/shared/config"
	"github.com/tnqbao/gau-upload-service/shared/infra"
	"github.com/tnqbao/gau-upload-service/shared/repository"

	"log"
)

func main() {
	err := godotenv.Load("/gau_upload/upload.env")
	if err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// Initialize configuration and infrastructure
	cfg := config.NewConfig()
	repo := repository.NewRepository(cfg)
	infra := infra.InitInfra(cfg)

	// Initialize controller with the new configuration and infrastructure
	ctrl := controller.NewController(cfg, repo, infra)

	router := routes.SetupRouter(ctrl)
	router.Run(":8080")
}
