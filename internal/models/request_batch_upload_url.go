package models

// BatchUploadURLRequest adalah request body untuk endpoint POST /api/v1/upload-url/batch.
// Mendukung upload hingga 20 video sekaligus.
type BatchUploadURLRequestItem struct {
	Type  string `json:"type"  binding:"required,oneof=letter word"`
	Label string `json:"label" binding:"required"`
}

type BatchUploadURLRequest struct {
	Items []BatchUploadURLRequestItem `json:"items" binding:"required,min=1,max=20"`
}

// BatchUploadURLResponseItem adalah satu item hasil generate upload URL.
type BatchUploadURLResponseItem struct {
	SampleID  string `json:"sample_id"`
	VideoPath string `json:"video_path"`
	VideoURL  string `json:"video_url"`
	UploadURL string `json:"upload_url"`
}
