package models

// SampleItem merepresentasikan satu gesture (letter/word) beserta videonya.
type SampleItem struct {
	GestureType string  `json:"gesture_type"`
	GestureName string  `json:"gesture_name"`
	Videos      []Video `json:"videos"`
}

// SampleResponse adalah response untuk endpoint GET /api/v1/sample.
type SampleResponse struct {
	Letters []SampleItem `json:"letters"`
	Words   []SampleItem `json:"words"`
}
