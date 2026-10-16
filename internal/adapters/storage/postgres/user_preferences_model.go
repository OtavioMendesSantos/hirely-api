package postgres

import "time"

type UserPreferencesModel struct {
	UserID          string     `gorm:"type:uuid;primaryKey"`
	User            *UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Theme           string     `gorm:"type:varchar(10);not null;default:system;check:user_preferences_theme_check,theme IN ('light','dark','system')"`
	AutoArchiveDays int        `gorm:"type:integer;not null;default:60;check:user_preferences_auto_archive_days_check,auto_archive_days IN (0,30,60,90)"`
	ConfettiEnabled bool       `gorm:"not null;default:true"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
}

func (UserPreferencesModel) TableName() string { return "user_preferences" }
