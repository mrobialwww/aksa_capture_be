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
	videoRepo      *repository.VideoRepository
	r2Service      *services.R2Service
	videoProcessor *services.VideoProcessor
	publicURL      string
}

func NewVideoHandler(
	videoRepo *repository.VideoRepository,
	r2Service *services.R2Service,
	publicURL string,
) *VideoHandler {
	return &VideoHandler{
		videoRepo:      videoRepo,
		r2Service:      r2Service,
		videoProcessor: services.NewVideoProcessor(),
		publicURL:      publicURL,
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

// POST /api/v1/videos/batch
// Membuat metadata untuk banyak video sekaligus (maksimal 20).
// Setiap item diproses secara berurutan; jika satu gagal, item lain tetap diproses.
// Response berisi per-item status "success" atau "error".
func (h *VideoHandler) BatchCreateVideo(c *gin.Context) {
	var req models.BatchCreateVideoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	results := make([]models.BatchCreateVideoResult, 0, len(req.Items))
	hasError := false

	for i := range req.Items {
		item := &req.Items[i]

		// Tentukan task_type sesuai logic: is_correct true -> ["lr","vlm"], false -> ["vlm"]
		if item.Label.IsCorrect {
			item.TaskType = []string{"lr", "vlm"}
		} else {
			item.TaskType = []string{"vlm"}
		}

		err := h.videoRepo.Create(c.Request.Context(), *item)
		if err != nil {
			hasError = true
			results = append(results, models.BatchCreateVideoResult{
				SampleID: item.SampleID,
				Status:   "error",
				Message:  err.Error(),
			})
		} else {
			results = append(results, models.BatchCreateVideoResult{
				SampleID: item.SampleID,
				Status:   "success",
			})
		}
	}

	statusCode := http.StatusCreated
	if hasError {
		statusCode = http.StatusMultiStatus // 207 — partial success
	}

	c.JSON(statusCode, gin.H{"results": results})
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

// GET /api/v1/sample
// Mengambil 5 video sample untuk setiap huruf (a-z) dan setiap kata dari daftar kata yang ditentukan.
func (h *VideoHandler) GetSample(c *gin.Context) {
	const sampleLimit = 5

	letters := []string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
		"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}

	words := []string{
		"selamat pagi", "selamat siang", "selamat sore", "selamat malam",
		"aku", "saya", "kamu", "dari", "mana", "berasal",
		"halo", "kabar", "apa", "siapa", "perkenalkan",
		"nama", "sayang", "marah",
	}

	// Fetch letter samples
	letterVideos, err := h.videoRepo.FindSample(c.Request.Context(), "letter", letters, sampleLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	// Fetch word samples
	wordVideos, err := h.videoRepo.FindSample(c.Request.Context(), "word", words, sampleLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	// Group letter videos by gesture_name
	letterMap := make(map[string]*models.SampleItem)
	for _, letter := range letters {
		letterMap[letter] = &models.SampleItem{
			GestureType: "letter",
			GestureName: letter,
			Videos:      []models.Video{},
		}
	}
	for _, v := range letterVideos {
		if item, ok := letterMap[v.Label.GestureName]; ok {
			item.Videos = append(item.Videos, v)
		}
	}

	// Group word videos by gesture_name
	wordMap := make(map[string]*models.SampleItem)
	for _, word := range words {
		wordMap[word] = &models.SampleItem{
			GestureType: "word",
			GestureName: word,
			Videos:      []models.Video{},
		}
	}
	for _, v := range wordVideos {
		if item, ok := wordMap[v.Label.GestureName]; ok {
			item.Videos = append(item.Videos, v)
		}
	}

	// Build ordered slices (preserving the original order)
	letterItems := make([]models.SampleItem, 0, len(letters))
	for _, letter := range letters {
		letterItems = append(letterItems, *letterMap[letter])
	}

	wordItems := make([]models.SampleItem, 0, len(words))
	for _, word := range words {
		wordItems = append(wordItems, *wordMap[word])
	}

	c.JSON(http.StatusOK, models.SampleResponse{
		Letters: letterItems,
		Words:   wordItems,
	})
}

// POST /api/v1/upload-url/batch
// Generate upload URL untuk banyak video sekaligus (maksimal 20).
func (h *VideoHandler) BatchGenerateUploadURL(c *gin.Context) {
	var req models.BatchUploadURLRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	baseURL := h.publicURL
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}

	results := make([]models.BatchUploadURLResponseItem, 0, len(req.Items))

	for _, item := range req.Items {
		sampleID := uuid.New().String()

		// Format: Dataset/{type}/{label}/record_{sampleID}.mp4
		// Menggunakan sampleID (UUID) bukan timestamp agar unik meski diproses dalam loop cepat.
		videoPath := fmt.Sprintf(
			"Dataset/%s/%s/record_%s.mp4",
			item.Type,
			item.Label,
			sampleID,
		)

		uploadURL, err := h.r2Service.GenerateUploadURL(videoPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": fmt.Sprintf("failed to generate upload URL for %s/%s: %s", item.Type, item.Label, err.Error()),
			})
			return
		}

		videoURL := baseURL + videoPath

		results = append(results, models.BatchUploadURLResponseItem{
			SampleID:  sampleID,
			VideoPath: videoPath,
			VideoURL:  videoURL,
			UploadURL: uploadURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
	})
}

// POST /api/v1/videos/direct-upload
// Menerima multipart/form-data: video file dan metadata.
// Strip audio, upload ke R2, lalu simpan metadata.
func (h *VideoHandler) DirectUpload(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(30 << 20); err != nil { // 30 MB max
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to parse multipart form"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "video file is required"})
		return
	}
	defer file.Close()

	// Read file into memory (since mobile videos shouldn't be huge)
	fileData := make([]byte, header.Size)
	if _, err := file.Read(fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read video file"})
		return
	}

	// Read form values
	videoType := c.PostForm("type")
	label := c.PostForm("label")
	name := c.PostForm("name")
	gender := c.PostForm("gender")
	isCorrectStr := c.PostForm("is_correct")
	errorCategory := c.PostForm("error_category")
	captureLocation := c.PostForm("capture_location")

	isCorrect := false
	if isCorrectStr == "true" {
		isCorrect = true
	}

	// 1. Strip audio using go-mp4 (only for MP4)
	processedData := fileData
	// We'll attempt to strip audio. If it fails (e.g. not MP4), we just proceed with original data.
	mutedData, err := h.videoProcessor.StripAudio(fileData)
	if err == nil && len(mutedData) > 0 {
		processedData = mutedData
	}

	// 2. Generate Upload URL or upload directly via R2 service
	sampleID := uuid.New().String()
	videoPath := fmt.Sprintf("Dataset/%s/%s/record_%s.mp4", videoType, label, sampleID)
	
	// Since we need to put the object, we should ideally have a PutObject method in R2Service
	// But wait, the client is already in R2Service. Let's just generate URL and upload, OR use s3.Client directly.
	// We can add UploadVideo(key, data, contentType) to R2Service.
	// For now, let's use the R2Client if accessible, or we'll need to modify R2Service.
	
	// I'll add UploadVideo to R2Service next.
	err = h.r2Service.UploadVideo(c.Request.Context(), videoPath, processedData, "video/mp4")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upload video to R2: " + err.Error()})
		return
	}

	baseURL := h.publicURL
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	videoURL := baseURL + videoPath

	// 3. Save metadata
	taskType := []string{"vlm"}
	if isCorrect {
		taskType = []string{"lr", "vlm"}
	}

	var errCat *string
	if errorCategory != "" {
		errCat = &errorCategory
	}

	req := models.CreateVideoRequest{
		SampleID: sampleID,
		TaskType: taskType,
	}
	req.Media.VideoPath = videoPath
	req.Media.VideoURL = videoURL
	req.Media.CaptureLocation = captureLocation

	req.Label.GestureType = videoType
	req.Label.GestureName = label
	req.Label.IsCorrect = isCorrect
	req.Label.ErrorCategory = errCat
	// Assigning default region if empty because it might be required by validator.
	req.Label.BisindoRegionVersion.Region = "Unknown"
	req.Label.BisindoRegionVersion.Subregion = "Unknown"

	req.Signer.SignerName = name
	req.Signer.Gender = gender

	err = h.videoRepo.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save metadata: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "success",
		"sample_id": sampleID,
	})
}
