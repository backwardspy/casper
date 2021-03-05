package models

import "gorm.io/gorm"

// MeatballDay represents a user's birth day and month.
type MeatballDay struct {
	gorm.Model
	GuildID string `gorm:"index:idx_unique_guild_member,unique"`
	UserID  string `gorm:"index:idx_unique_guild_member,unique"`
	Month   uint
	Day     uint
}

// MeatballRole maps guild IDs to their respective meatball day roles.
type MeatballRole struct {
	gorm.Model
	GuildID string `gorm:"uniqueIndex"`
	RoleID  string
}

// MeatballChannel maps guild IDs to their respective announcement channels.
type MeatballChannel struct {
	gorm.Model
	GuildID   string `gorm:"uniqueIndex"`
	ChannelID string
}
