package models

type CreateVideoRequest struct {
	ID        string `json:"id"`
	VideoPath string `json:"video_path"`

	Name   string `json:"name" binding:"required"`
	Gender string `json:"gender" binding:"required,oneof=male female"`

	Label string `json:"label"`
	Type  string `json:"type"`

	IsCorrect bool   `json:"is_correct"`
	Notes     string `json:"notes"`
}
