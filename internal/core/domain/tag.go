package domain

import "time"

type Tag struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	ColorHex  string    `json:"colorHex"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewTag(id, userID, name, colorHex string) *Tag {
	return &Tag{
		ID:        id,
		UserID:    userID,
		Name:      name,
		ColorHex:  colorHex,
		CreatedAt: time.Now().UTC(),
	}
}
