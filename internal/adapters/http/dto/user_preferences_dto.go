package dto

type UpdateUserPreferencesRequest struct {
	AutoArchiveDays *int    `json:"auto_archive_days"`
	Theme           *string `json:"theme"`
	ConfettiEnabled *bool   `json:"confetti_enabled"`
}
