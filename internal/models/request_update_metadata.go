package models

type UpdateMetadataRequest struct {
	// Label fields
	ErrorCategory *string `json:"error_category" binding:"omitempty,oneof=handshape_wrong orientation_wrong location_wrong movement_wrong non_manual_marker_missing finger_spelling_incomplete mixed_with_other_sign unclear"`
	ValidatedBy   *string `json:"validated_by"`
	Reasoning     *string `json:"reasoning"`

	// Quality fields
	HandsVisible *bool `json:"hands_visible"`
	FaceVisible  *bool `json:"face_visible"`
	HandsClear   *bool `json:"hands_clear"`
	FaceClear    *bool `json:"face_clear"`
}
