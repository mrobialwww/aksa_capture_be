package models

type Meta struct {
	CurrentPage int `json:"current_page"`
	Limit       int `json:"limit"`
	TotalItems  int `json:"total_items"`
	TotalPages  int `json:"total_pages"`
}

type PaginatedResponse struct {
	Data []Video `json:"data"`
	Meta Meta    `json:"meta"`
}
