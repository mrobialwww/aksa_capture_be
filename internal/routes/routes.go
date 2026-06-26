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

		// Batch create Video metadata (max 20)
		// POST api/v1/videos/batch
		api.POST(
			"/videos/batch",
			videoHandler.BatchCreateVideo,
		)

		// Direct upload with audio stripping
		// POST api/v1/videos/direct-upload
		api.POST(
			"/videos/direct-upload",
			videoHandler.DirectUpload,
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

		// GET /api/v1/sample
		// Mengambil 5 video per huruf (a-z) dan 5 video per kata dari daftar kata
		api.GET(
			"/sample",
			videoHandler.GetSample,
		)

		// POST /api/v1/upload-url/batch
		// Generate upload URL untuk banyak video sekaligus (max 20)
		api.POST(
			"/upload-url/batch",
			videoHandler.BatchGenerateUploadURL,
		)
	}
}
