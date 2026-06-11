package models

// GenerateUploadURLRequest adalah body untuk POST /api/v1/upload-url
type GenerateUploadURLRequest struct {
	Type  string `json:"type"  binding:"required,oneof=letter word"`
	Label string `json:"label" binding:"required"`
}
