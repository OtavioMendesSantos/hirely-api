package dto

import "hirely-api/internal/core/domain"

type CreateTagRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	ColorHex string `json:"color_hex" binding:"required,hexcolor"`
}

type ListTagsResponse struct {
	Tags []*domain.Tag `json:"tags"`
}
