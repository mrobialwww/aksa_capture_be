package handlers

import (
	"aksa_capture_be/internal/models"
	"aksa_capture_be/internal/repository"
	"aksa_capture_be/internal/services"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type VideoHandler struct {
	videoRepo *repository.VideoRepository
	r2Service *services.R2Service
	publicURL string
}

func NewVideoHandler(
	videoRepo *repository.VideoRepository,
	r2Service *services.R2Service,
	publicURL string,
) *VideoHandler {
	return &VideoHandler{
		videoRepo: videoRepo,
		r2Service: r2Service,
		publicURL: publicURL,
	}
}

// POST /api/v1/upload-url
func (h *VideoHandler) GenerateUploadURL(
	c *gin.Context,
) {
	var req models.GenerateUploadURLRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"message": err.Error()},
		)
		return
	}

	id := uuid.New().String()
	timestamp := time.Now().UnixMilli()

	// Format: Dataset/{type}/{label}/record_{timestamp}.mp4
	videoPath := fmt.Sprintf(
		"Dataset/%s/%s/record_%d.mp4",
		req.Type,
		req.Label,
		timestamp,
	)

	uploadURL, err :=
		h.r2Service.GenerateUploadURL(
			videoPath,
		)

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"id":         id,
			"video_path": videoPath,
			"upload_url": uploadURL,
		},
	)
}

// POST /api/v1/videos
func (h *VideoHandler) CreateVideo(
	c *gin.Context,
) {

	var req models.CreateVideoRequest

	if err := c.ShouldBindJSON(
		&req,
	); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": err.Error(),
			},
		)

		return
	}

	video := models.Video{
		ID:        req.ID,
		VideoPath: req.VideoPath,
		Label:     req.Label,
		Type:      req.Type,
		IsCorrect: req.IsCorrect,
		Notes:     req.Notes,
	}

	err := h.videoRepo.Create(
		video,
	)

	if err != nil {

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"message": "video metadata created",
		},
	)
}

// GET /api/v1/videos
// Supports optional query params: is_correct, type, label
// If none provided, returns all videos.
func (h *VideoHandler) GetVideos(
	c *gin.Context,
) {
	filter := models.VideoFilter{}

	hasFilter := false

	if isCorrectStr := c.Query("is_correct"); isCorrectStr != "" {
		val, err := strconv.ParseBool(isCorrectStr)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "is_correct must be true or false"},
			)
			return
		}
		filter.IsCorrect = &val
		hasFilter = true
	}

	if typeStr := c.Query("type"); typeStr != "" {
		if typeStr != "huruf" && typeStr != "kata" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "type must be 'huruf' or 'kata'"},
			)
			return
		}
		filter.Type = typeStr
		hasFilter = true
	}

	if labelStr := c.Query("label"); labelStr != "" {
		filter.Label = labelStr
		hasFilter = true
	}

	var (
		videos []models.Video
		err    error
	)

	if hasFilter {
		videos, err = h.videoRepo.FindByFilter(filter)
	} else {
		videos, err = h.videoRepo.FindAll()
	}

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	for i := range videos {
		baseURL := h.publicURL
		if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
			baseURL += "/"
		}
		videos[i].VideoURL = baseURL + videos[i].VideoPath
	}

	c.JSON(
		http.StatusOK,
		gin.H{"data": videos},
	)
}

// GET /api/v1/videos/:id
func (h *VideoHandler) GetVideoByID(
	c *gin.Context,
) {
	id := c.Param("id")

	video, err := h.videoRepo.FindByID(id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(
				http.StatusNotFound,
				gin.H{"message": "video not found"},
			)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	baseURL := h.publicURL
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	video.VideoURL = baseURL + video.VideoPath

	c.JSON(
		http.StatusOK,
		gin.H{"data": video},
	)
}

// PATCH /api/v1/videos/:id/notes
func (h *VideoHandler) UpdateNotes(
	c *gin.Context,
) {
	id := c.Param("id")

	var req models.UpdateNotesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"message": err.Error()},
		)
		return
	}

	err := h.videoRepo.UpdateNotes(id, req.Notes)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(
				http.StatusNotFound,
				gin.H{"message": "video not found"},
			)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{"message": "notes updated"},
	)
}
