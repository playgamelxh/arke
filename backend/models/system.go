package models

import "time"

type SystemSetting struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SettingKey   string    `json:"settingKey" gorm:"uniqueIndex"`
	SettingValue string    `json:"settingValue" gorm:"type:text"`
	UpdatedAt    time.Time `json:"updatedAt"`
}