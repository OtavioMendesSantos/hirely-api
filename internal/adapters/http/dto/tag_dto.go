package dto

import "hirely-api/internal/core/domain"

type CreateTagRequest struct {
	Name     string `json:"name"`
	ColorHex string `json:"color_hex"`
}

type ListTagsResponse struct {
	Tags []*domain.Tag `json:"tags"`
}
