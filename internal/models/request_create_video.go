package models

type CreateVideoRequest struct {
	SampleID string   `json:"sample_id" binding:"required"`
	TaskType []string `json:"task_type"` // Akan di-override berdasarkan is_correct

	Media struct {
		VideoPath   string  `json:"video_path" binding:"required"`
		VideoURL    string  `json:"video_url" binding:"required"`
		DurationSec float64 `json:"duration_sec"`
		Resolution  struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"resolution"`
		CaptureLocation string `json:"capture_location" binding:"omitempty,oneof=indoor outdoor"`
	} `json:"media"`

	Label struct {
		GestureType          string `json:"gesture_type" binding:"required,oneof=letter word"`
		GestureName          string `json:"gesture_name" binding:"required"`
		BisindoRegionVersion struct {
			Region    string `json:"region"    binding:"required"`
			Subregion string `json:"subregion" binding:"required"`
		} `json:"bisindo_region_version"`
		IsCorrect     bool    `json:"is_correct"`
		ErrorCategory *string `json:"error_category"`
		ValidatedBy   *string `json:"validated_by"`
		Reasoning     *string `json:"reasoning"`
	} `json:"label"`

	// Wajib diisi: informasi penandatangan
	Signer struct {
		SignerName string `json:"signer_name" binding:"required"`
		Gender     string `json:"gender"      binding:"required,oneof=male female"`
	} `json:"signer"`

	Quality struct {
		HandsVisible bool `json:"hands_visible"`
		FaceVisible  bool `json:"face_visible"`
		HandsClear   bool `json:"hands_clear"`
		FaceClear    bool `json:"face_clear"`
	} `json:"quality"`
}
