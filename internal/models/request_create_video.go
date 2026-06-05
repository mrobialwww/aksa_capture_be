package models

type CreateVideoRequest struct {
	ID        string `json:"id"`
	VideoPath string `json:"video_path"`

	Label string `json:"label"`
	Type  string `json:"type"`

	IsCorrect bool   `json:"is_correct"`
	Notes     string `json:"notes"`
}
