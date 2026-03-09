package models

type VideoJob struct {
	VideoID   string `json:"video_id"`
	InputURL  string `json:"input_url"`
	OutputURL string `json:"output_url,omitempty"`
}
