package models

import "time"

// Video adalah representasi lengkap dari satu sample video (JOIN semua tabel).
type Video struct {
	SampleID  string    `json:"sample_id"`
	TaskType  []string  `json:"task_type"`
	CreatedAt time.Time `json:"created_at"`

	Media   VideoMedia   `json:"media"`
	Label   VideoLabel   `json:"label"`
	Signer  VideoSigner  `json:"signer"`
	Quality VideoQuality `json:"quality"`
}

// VideoMedia merepresentasikan tabel media.
type VideoMedia struct {
	VideoPath        string  `json:"video_path"`
	VideoURL         string  `json:"video_url,omitempty"`
	DurationSec      float64 `json:"duration_sec"`
	ResolutionWidth  int     `json:"resolution_width"`
	ResolutionHeight int     `json:"resolution_height"`
	CaptureLocation  string  `json:"capture_location"`
}

// VideoLabel merepresentasikan tabel label.
type VideoLabel struct {
	GestureType      string `json:"gesture_type"`
	GestureName      string `json:"gesture_name"`
	TargetID         string `json:"target_id"`
	BisindoRegion    string `json:"bisindo_region"`
	BisindoSubregion string `json:"bisindo_subregion"`
	IsCorrect        bool   `json:"is_correct"`
	ErrorCategory    string `json:"error_category,omitempty"`
	ValidatedBy      string `json:"validated_by,omitempty"`
	Reasoning        string `json:"reasoning,omitempty"`
}

// VideoSigner merepresentasikan tabel signer.
type VideoSigner struct {
	SignerName string `json:"signer_name"`
	Gender     string `json:"gender"`
}

// VideoQuality merepresentasikan tabel quality.
type VideoQuality struct {
	HandsVisible bool `json:"hands_visible"`
	FaceVisible  bool `json:"face_visible"`
	HandsClear   bool `json:"hands_clear"`
	FaceClear    bool `json:"face_clear"`
}
