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
		// Upload
		api.POST(
			"/upload-url",
			videoHandler.GenerateUploadURL,
		)

		// Videos
		api.POST(
			"/videos",
			videoHandler.CreateVideo,
		)

		// GET /api/v1/videos
		// Query params (all optional, combinable):
		//   ?is_correct=true|false
		//   ?type=huruf|kata
		//   ?label=<string>
		api.GET(
			"/videos",
			videoHandler.GetVideos,
		)

		// GET /api/v1/videos/:id
		api.GET(
			"/videos/:id",
			videoHandler.GetVideoByID,
		)

		// PATCH /api/v1/videos/:id/notes
		api.PATCH(
			"/videos/:id/notes",
			videoHandler.UpdateNotes,
		)
	}
}
