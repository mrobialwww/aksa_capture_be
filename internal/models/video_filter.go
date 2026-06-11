package models

// VideoFilter holds optional query parameters for filtering videos.
// A nil pointer or empty string means the field is not applied as a filter.
type VideoFilter struct {
	IsCorrect  *bool  // nil = no filter
	Type       string // "" = no filter; valid: "letter", "word"
	Label      string // "" = no filter; partial match (ILIKE)
	SignerName string // "" = no filter; partial match (ILIKE) on signer.signer_name
	Page       int    // pagination
	Limit      int    // pagination
}
