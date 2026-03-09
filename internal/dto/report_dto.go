package dto

type ReportVideoRequest struct {
	Reason string `json:"reason" binding:"required"`
}
