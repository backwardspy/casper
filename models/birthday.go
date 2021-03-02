package models

import "gorm.io/gorm"

// MeatballDay represents a user's birth day and month
type MeatballDay struct {
	gorm.Model
	UserID string `gorm:"uniqueIndex"`
	Month  uint
	Day    uint
}
