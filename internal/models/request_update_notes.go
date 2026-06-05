package models

// UpdateNotesRequest adalah body untuk PATCH /api/v1/videos/:id/notes
type UpdateNotesRequest struct {
	Notes string `json:"notes" binding:"required"`
}
