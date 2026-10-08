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
		// ── Explicit OPTIONS for every POST/PATCH/DELETE endpoint ──────────────
		// Wajib ada agar browser bisa menyelesaikan CORS preflight tanpa ambigu di routing Gin.
		api.OPTIONS("/upload-url", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/upload-url/batch", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/videos", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/videos/:id", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/videos/batch", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/videos/:id/metadata", func(c *gin.Context) { c.Status(204) })
		api.OPTIONS("/sample", func(c *gin.Context) { c.Status(204) })

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
		api.GET(
			"/sample",
			videoHandler.GetSample,
		)

		// POST /api/v1/upload-url/batch
		api.POST(
			"/upload-url/batch",
			videoHandler.BatchGenerateUploadURL,
		)
	}
}
