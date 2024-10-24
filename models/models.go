package models

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	UserID                 string    `gorm:"index"` // The ID of the user.
	ChannelID              string    // The ID of the channel associated with the account.
	Title                  string    // The title of the account.
	LastStatus             Status    `gorm:"default:unknown"` // The last known status of the account.
	LastCheck              int64     `gorm:"default:0"`       // The timestamp of the last check performed on the account.
	LastNotification       int64     // The timestamp of the last daily notification sent out on the account.
	LastCookieNotification int64     // The timestamp of the last notification sent out on the account for an expired ssocookie.
	SSOCookie              string    // The SSO cookie associated with the account.
	Created                int64     // The timestamp of when the account was created on Activision.
	IsExpiredCookie        bool      `gorm:"default:false"`   // A flag indicating if the SSO cookie has expired.
	NotificationType       string    `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	IsPermabanned          bool      `gorm:"default:false"`   // A flag indicating if the account is permanently banned
	LastCookieCheck        int64     `gorm:"default:0"`       // The timestamp of the last cookie check for permanently banned accounts.
	LastStatusChange       int64     `gorm:"default:0"`       // The timestamp of the last status change
	IsCheckDisabled        bool      `gorm:"default:false"`   // A flag indicating if checks are disabled for this account
	DisabledReason         string    // Reason for disabling checks
	SSOCookieExpiration    int64     // The timestamp of the SSO cookie expiration
	ConsecutiveErrors      int       `gorm:"default:0"` // The number of consecutive errors encountered while checking the account
	LastSuccessfulCheck    time.Time // The timestamp of the last successful check
	LastErrorTime          time.Time // The timestamp of the last error encountered
	Last24HourNotification time.Time // The timestamp of the last 24-hour notification
}

type UserSettings struct {
	gorm.Model
	UserID                       string    `gorm:"uniqueIndex"`
	EZCaptchaAPIKey              string    // User's own EZCaptcha API key, if provided
	TwoCaptchaAPIKey             string    // User's own 2captcha API key, if provided
	PreferredCaptchaProvider     string    `gorm:"default:'ezcaptcha'"` // 'ezcaptcha' or '2captcha'
	CaptchaBalance               float64   // Current balance for the selected provider
	LastBalanceCheck             time.Time // Last time the balance was checked
	CheckInterval                int       // In minutes
	NotificationInterval         float64   // In hours
	CooldownDuration             float64   // In hours
	StatusChangeCooldown         float64   // In hours
	HasSeenAnnouncement          bool      `gorm:"default:false"`   // Flag to track if the user has seen the global announcement.
	NotificationType             string    `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	LastNotification             time.Time // Timestamp of the last notification
	LastDisabledNotification     time.Time // Timestamp of the last disabled notification
	LastStatusChangeNotification time.Time // Timestamp of the last status change notification
	LastDailyUpdateNotification  time.Time // Timestamp of the last daily update notification
	LastCookieExpirationWarning  time.Time // Timestamp of the last cookie expiration warning
	LastBalanceNotification      time.Time // Timestamp of the last balance notification
	LastErrorNotification        time.Time // Timestamp of the last error notification
	CustomSettings               bool      `gorm:"default:false"` // Flag to indicate if user has custom settings
}

type Ban struct {
	gorm.Model
	Account         Account // The account that has a status history.
	AccountID       uint    // The ID of the account.
	Status          Status  // The status of the ban.
	TempBanDuration string  // Duration of the temporary ban (if applicable)
	AffectedGames   string  // Comma-separated list of affected games
}

type Status string

const (
	StatusGood          Status = "Good"           // The account status returned as good standing.
	StatusPermaban      Status = "Permaban"       // The account status returned as permanent ban.
	StatusShadowban     Status = "Shadowban"      // The account status returned as shadowban.
	StatusUnknown       Status = "Unknown"        // The account status not known.
	StatusInvalidCookie Status = "Invalid_Cookie" // The account has an invalid SSO cookie.
	StatusTempban       Status = "Temporary"      // The account status returned as temporarily banned.
)

type CaptchaProvider string

const (
	EZCaptcha  CaptchaProvider = "ezcaptcha"
	TwoCaptcha CaptchaProvider = "2captcha"
)

type CaptchaBalance struct {
	Provider  CaptchaProvider
	Balance   float64
	CheckTime time.Time
}
