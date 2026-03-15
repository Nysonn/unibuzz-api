package dto

type ReportReason string

const (
	ReportReasonSelfHarm             ReportReason = "self_harm"
	ReportReasonHarassment           ReportReason = "harassment"
	ReportReasonInappropriateContent ReportReason = "inappropriate_content"
	ReportReasonSpam                 ReportReason = "spam"
	ReportReasonOther                ReportReason = "other"
)

func (r ReportReason) Valid() bool {
	switch r {
	case ReportReasonSelfHarm, ReportReasonHarassment, ReportReasonInappropriateContent, ReportReasonSpam, ReportReasonOther:
		return true
	}
	return false
}

type ReportVideoRequest struct {
	Reason       ReportReason `json:"reason" binding:"required"`
	CustomReason string       `json:"custom_reason" binding:"omitempty,max=100"`
}
