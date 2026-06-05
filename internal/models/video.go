package models

import "time"

type Video struct {
	ID        string    `json:"id"`
	VideoPath string    `json:"video_path"`
	VideoURL  string    `json:"video_url"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`
	IsCorrect bool      `json:"is_correct"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}
