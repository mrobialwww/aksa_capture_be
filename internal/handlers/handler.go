package handlers

import (
	"aksa_capture_be/internal/models"
	"aksa_capture_be/internal/repository"
	"aksa_capture_be/internal/services"
	"errors"
	"fmt"
	"math"
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

	sampleID := uuid.New().String()
	timestamp := time.Now().UnixMilli()

	// Format: Dataset/{type}/{label}/record_{timestamp}.mp4
	videoPath := fmt.Sprintf(
		"Dataset/%s/%s/record_%d.mp4",
		req.Type,
		req.Label,
		timestamp,
	)

	uploadURL, err := h.r2Service.GenerateUploadURL(videoPath)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	// Bangun public URL final video setelah di-upload
	baseURL := h.publicURL
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	videoURL := baseURL + videoPath

	c.JSON(
		http.StatusOK,
		gin.H{
			"sample_id":  sampleID,
			"video_path": videoPath,
			"video_url":  videoURL,
			"upload_url": uploadURL,
		},
	)
}

// POST /api/v1/videos
func (h *VideoHandler) CreateVideo(
	c *gin.Context,
) {
	var req models.CreateVideoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"message": err.Error()},
		)
		return
	}

	// Tentukan task_type sesuai logic:
	// jika is_correct true -> ["lr", "vlm"], jika false -> ["vlm"]
	if req.Label.IsCorrect {
		req.TaskType = []string{"lr", "vlm"}
	} else {
		req.TaskType = []string{"vlm"}
	}

	// Execute creation in repository using request context
	err := h.videoRepo.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{"message": "video metadata created"},
	)
}

// GET /api/v1/videos
// Supports optional query params: is_correct, type, label
func (h *VideoHandler) GetVideos(
	c *gin.Context,
) {
	var filter models.VideoFilter

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := 40
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	filter.Page = page
	filter.Limit = limit

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
	}

	if typeStr := c.Query("type"); typeStr != "" {
		if typeStr != "letter" && typeStr != "word" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"message": "type must be 'letter' or 'word'"},
			)
			return
		}
		filter.Type = typeStr
	}

	if labelStr := c.Query("label"); labelStr != "" {
		filter.Label = labelStr
	}

	if signerNameStr := c.Query("signer_name"); signerNameStr != "" {
		filter.SignerName = signerNameStr
	}

	videos, totalItems, err := h.videoRepo.FindByFilter(c.Request.Context(), filter)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"message": err.Error()},
		)
		return
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	c.JSON(
		http.StatusOK,
		models.PaginatedResponse{
			Data: videos,
			Meta: models.Meta{
				CurrentPage: page,
				Limit:       limit,
				TotalItems:  totalItems,
				TotalPages:  totalPages,
			},
		},
	)
}

// GET /api/v1/videos/:id
func (h *VideoHandler) GetVideoByID(
	c *gin.Context,
) {
	id := c.Param("id")

	video, err := h.videoRepo.FindByID(c.Request.Context(), id)
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
		gin.H{"data": video},
	)
}

// PATCH /api/v1/videos/:id/metadata
func (h *VideoHandler) UpdateMetadata(
	c *gin.Context,
) {
	id := c.Param("id")

	var req models.UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"message": err.Error()},
		)
		return
	}

	err := h.videoRepo.UpdateMetadata(c.Request.Context(), id, req)
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
		gin.H{"message": "video review updated"},
	)
}

// DELETE /api/v1/videos/:id
// Menghapus metadata dari semua tabel DB dan file dari R2 Cloudflare.
func (h *VideoHandler) DeleteVideo(
	c *gin.Context,
) {
	id := c.Param("id")

	// 1. Hapus dari DB, dapatkan video_path untuk delete dari R2
	videoPath, err := h.videoRepo.Delete(c.Request.Context(), id)
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

	// 2. Hapus file dari R2 (hanya jika video_path tidak kosong)
	if videoPath != "" {
		if err := h.r2Service.DeleteObject(c.Request.Context(), videoPath); err != nil {
			// DB sudah terhapus; log error R2 tapi tetap return 200 dengan peringatan
			c.JSON(
				http.StatusOK,
				gin.H{
					"message":  "video metadata deleted, but failed to delete file from R2",
					"r2_error": err.Error(),
				},
			)
			return
		}
	}

	c.JSON(
		http.StatusOK,
		gin.H{"message": "video deleted successfully"},
	)
}
