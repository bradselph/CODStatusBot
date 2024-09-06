package models

import (
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	GuildID                string `gorm:"index"` // The ID of the guild the account belongs to.
	UserID                 string `gorm:"index"` // The ID of the user.
	ChannelID              string // The ID of the channel associated with the account.
	Title                  string // The title of the account.
	LastStatus             Status `gorm:"default:unknown"` // The last known status of the account.
	LastCheck              int64  `gorm:"default:0"`       // The timestamp of the last check performed on the account.
	LastNotification       int64  // The timestamp of the last daily notification sent out on the account.
	LastCookieNotification int64  // The timestamp of the last notification sent out on the account for an expired ssocookie.
	SSOCookie              string // The SSO cookie associated with the account.
	SSOCookieExpiration    int64
	Created                string // The timestamp of when the account was created on Activision.
	IsExpiredCookie        bool   `gorm:"default:false"`   // A flag indicating if the SSO cookie has expired.
	NotificationType       string `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	IsPermabanned          bool   `gorm:"default:false"`   // A flag indicating if the account is permanently banned
	LastCookieCheck        int64  `gorm:"default:0"`       // The timestamp of the last cookie check for permanently banned accounts
	LastStatusChange       int64  `gorm:"default:0"`       // The timestamp of the last status change
	IsCheckDisabled        bool   `gorm:"default:false"`   // A flag indicating if checks are disabled for this account
}

type UserSettings struct {
	gorm.Model
	UserID               string  `gorm:"uniqueIndex"`
	CaptchaAPIKey        string  // User's own API key, if provided
	CheckInterval        int     // in minutes
	NotificationInterval float64 // in hours
	CooldownDuration     float64 // in hours
	StatusChangeCooldown float64 // in hours
	NotificationType     string  `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	HasSeenAnnouncement  bool    `gorm:"default:false"`   // Flag to track if the user has seen the global announcement
}

type Ban struct {
	gorm.Model
	Account   Account // The account that has been banned.
	AccountID uint    // The ID of the banned account.
	Status    Status  // The status of the ban.
}

type Status string

const (
	StatusGood          Status = "good"
	StatusPermaban      Status = "permaban"
	StatusShadowban     Status = "shadowban"
	StatusUnknown       Status = "unknown"
	StatusInvalidCookie Status = "invalid_cookie"
	StatusTempban       Status = "temporary"
)
