package models

// BatchCreateVideoRequest adalah request body untuk POST /api/v1/videos/batch.
// Mendukung pembuatan metadata hingga 100 video sekaligus.
type BatchCreateVideoRequest struct {
	Items []CreateVideoRequest `json:"items" binding:"required,min=1,max=100"`
}

// BatchCreateVideoResult adalah hasil untuk satu item dalam batch create.
type BatchCreateVideoResult struct {
	SampleID string `json:"sample_id"`
	Status   string `json:"status"`           // "success" atau "error"
	Message  string `json:"message,omitempty"` // diisi jika error
}
