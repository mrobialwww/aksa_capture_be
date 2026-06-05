package main

import (
	"log"
	"os"

	"aksa_capture_be/internal/config"
	"aksa_capture_be/internal/database"
	"aksa_capture_be/internal/handlers"
	"aksa_capture_be/internal/repository"
	"aksa_capture_be/internal/routes"
	"aksa_capture_be/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found")
	}

	// Database
	db, err := database.NewDatabase(
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// R2
	r2Client, err := config.NewR2Client(
		os.Getenv("R2_ACCOUNT_ID"),
		os.Getenv("R2_ACCESS_KEY_ID"),
		os.Getenv("R2_SECRET_ACCESS_KEY"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Repository
	videoRepo := repository.NewVideoRepository(
		db,
	)

	// Service
	r2Service := services.NewR2Service(
		r2Client,
		os.Getenv("R2_BUCKET_NAME"),
	)

	// Handler
	videoHandler := handlers.NewVideoHandler(
		videoRepo,
		r2Service,
		os.Getenv("R2_PUBLIC_URL"),
	)

	// Router
	router := gin.Default()

	routes.RegisterRoutes(
		router,
		videoHandler,
	)

	if err := router.Run(
		":" + os.Getenv("PORT"),
	); err != nil {
		log.Fatal(err)
	}
}
