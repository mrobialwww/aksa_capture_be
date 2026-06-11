package routes

import (
	"github.com/gin-gonic/gin"

	"aksa_capture_be/internal/handlers"
)

func RegisterRoutes(
	router *gin.Engine,
	videoHandler *handlers.VideoHandler,
) {

	api := router.Group("/api/v1")

	{
		// Upload Video to Cloudflare to generated URL
		// POST api/v1/upload-url
		api.POST(
			"/upload-url",
			videoHandler.GenerateUploadURL,
		)

		// Create Video metadata
		// POST api/v1/videos
		api.POST(
			"/videos",
			videoHandler.CreateVideo,
		)

		// GET /api/v1/videos
		api.GET(
			"/videos",
			videoHandler.GetVideos,
		)

		// GET /api/v1/videos/:id
		api.GET(
			"/videos/:id",
			videoHandler.GetVideoByID,
		)

		// PATCH /api/v1/videos/:id/metadata
		api.PATCH(
			"/videos/:id/metadata",
			videoHandler.UpdateMetadata,
		)

		// DELETE /api/v1/videos/:id
		api.DELETE(
			"/videos/:id",
			videoHandler.DeleteVideo,
		)
	}
}
