package dto

type CreateCommentRequest struct {
	Comment string `json:"comment" binding:"required,max=500"`
}

type UpdateCommentRequest struct {
	Comment string `json:"comment" binding:"required,max=500"`
}
