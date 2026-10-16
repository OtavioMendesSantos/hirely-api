package dto

type UpdateUserPreferencesRequest struct {
	Theme           *string `json:"theme"`
	ConfettiEnabled *bool   `json:"confetti_enabled"`
}
