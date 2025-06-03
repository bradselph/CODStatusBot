package models

import (
	"time"

	"gorm.io/gorm"
)

type Account struct { // The accounts table
	gorm.Model
	UserID                  string            `gorm:"index"`      // The ID of the user.
	GuildID                 string            `gorm:"default:''"` // The guild ID if the account was added in a server context
	ChannelID               string            // The ID of the channel associated with the account.
	Title                   string            // user assigned title for the account
	ActivisionID            string            // The Activision ID associated with this account
	LastStatus              Status            `gorm:"default:unknown"` // The last known status of the account.
	LastCheck               int64             `gorm:"default:0"`       // The timestamp of the last check performed on the account.
	LastNotification        int64             // The timestamp of the last daily notification sent out on the account.
	LastCookieNotification  int64             // The timestamp of the last notification sent out on the account for an expired ssocookie.
	SSOCookie               string            // The SSO cookie associated with the account.
	Created                 int64             // The timestamp of when the account was created on Activision.
	IsExpiredCookie         bool              `gorm:"default:false"`   // A flag indicating if the SSO cookie has expired.
	NotificationType        string            `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	IsPermabanned           bool              `gorm:"default:false"`   // A flag indicating if the account is permanently banned
	IsShadowbanned          bool              `gorm:"default:false"`   // A flag indicating if the account is shadowbanned
	IsTempbanned            bool              `gorm:"default:false"`   // A flag indicating if the account is temporarily banned
	IsVIP                   bool              `gorm:"default:false"`   // A flag indicating if the account is a VIP
	IsOGVerdansk            bool              `gorm:"default:false"`   // A flag indicating if account has Verdansk stats
	LastCookieCheck         int64             `gorm:"default:0"`       // The timestamp of the last cookie check for permanently banned accounts.
	LastStatusChange        int64             `gorm:"default:0"`       // The timestamp of the last status change
	IsCheckDisabled         bool              `gorm:"default:false"`   // A flag indicating if checks are disabled for this account
	DisabledReason          string            // Reason for disabling checks
	SSOCookieExpiration     int64             // The timestamp of the SSO cookie expiration
	ConsecutiveErrors       int               `gorm:"default:0"` // The number of consecutive errors encountered while checking the account
	LastSuccessfulCheck     time.Time         // The timestamp of the last successful check
	LastErrorTime           time.Time         // The timestamp of the last error encountered
	Last24HourNotification  time.Time         // The timestamp of the last 24-hour notification
	LastCheckNowTime        time.Time         // For check now command rate limiting
	LastAddAccountTime      time.Time         // For add account rate limiting
	GameSpecificBans        map[string]string `gorm:"serializer:json"` // Map of game titles to ban status
	IsRankLocked            bool              `gorm:"default:false"`   // Flag for ranked play restriction
	IsCampaignOnlyShadowban bool              `gorm:"default:false"`   // Flag for BO6 campaign-only shadowbans
}

type UserSettings struct { // User settings for the bot
	gorm.Model
	UserID                       string               `gorm:"type:varchar(255);uniqueIndex"` // The ID of the user.
	CapSolverAPIKey              string               // User's own Capsolver API key, if provided
	EZCaptchaAPIKey              string               // User's own EZCaptcha API key, if provided
	TwoCaptchaAPIKey             string               // User's own 2captcha API key, if provided
	PreferredCaptchaProvider     string               `gorm:"default:'capsolver'"` // 'capsolver', 'ezcaptcha' or '2captcha'
	FallbackCaptchaProvider      string               `gorm:"default:''"`          // Fallback captcha provider when primary fails
	EnableFallback               bool                 `gorm:"default:true"`        // Enable fallback captcha when primary fails
	UseFallbackForDefault        bool                 `gorm:"default:true"`        // Use fallback for default key users
	CaptchaBalance               float64              // Current balance for the selected provider
	FallbackCaptchaBalance       float64              // Current balance for the fallback provider
	LastBalanceCheck             time.Time            // Last time the balance was checked
	CheckInterval                int                  // the user's set check interval
	NotificationInterval         float64              // the user's preferred notification interval
	CooldownDuration             float64              // the user's cooldown duration for actions
	StatusChangeCooldown         float64              // the user's cooldown duration for status changes
	HasSeenAnnouncement          bool                 `gorm:"default:false"`   // Flag to track if the user has seen the global announcement.
	NotificationType             string               `gorm:"default:channel"` // User preference for location of notifications either channel or dm
	PreferEphemeralResponses     bool                 `gorm:"default:true"`    // Flag to prefer ephemeral messages
	NotificationTimes            map[string]time.Time `gorm:"serializer:json"` // For all notification cooldowns
	ActionCounts                 map[string]int       `gorm:"serializer:json"` // For counting actions within time windows
	LastActionTimes              map[string]time.Time `gorm:"serializer:json"` // For tracking when actions were last performed
	LastNotification             time.Time            // Timestamp of the last notification
	LastDisabledNotification     time.Time            // Timestamp of the last disabled notification
	LastStatusChangeNotification time.Time            // Timestamp of the last status change notification
	LastDailyUpdateNotification  time.Time            // Timestamp of the last daily update notification
	LastCookieExpirationWarning  time.Time            // Timestamp of the last cookie expiration warning
	LastBalanceNotification      time.Time            // Timestamp of the last balance notification
	LastErrorNotification        time.Time            // Timestamp of the last error notification
	CustomSettings               bool                 `gorm:"default:false"`   // Flag to indicate if user has custom settings
	LastCommandTimes             map[string]time.Time `gorm:"serializer:json"` // Map of command names to their last execution time
	RateLimitExpiration          map[string]time.Time `gorm:"serializer:json"` // Map of command names to their rate limit expiration time
	InstallationType             string               `gorm:"default:''"`      // 'server' or 'direct' - how the user installed the bot
	InstallationGuildID          string               `gorm:"default:''"`      // The guild ID where the bot was installed (if server)
	InstallationTime             time.Time            // When the user first interacted with the bot
	LastGuildInteraction         time.Time            // Last time the user interacted in a guild context
	LastDirectInteraction        time.Time            // Last time the user interacted in DM context
	PrimaryInteractionContext    string               `gorm:"default:''"` // The context where the user interacts most frequently
	MessageFailures              int                  `gorm:"default:0"`  // Number of failed messages
	LastMessageFailure           time.Time            // Timestamp of the last failed message
	IsUnreachable                bool                 `gorm:"default:false"` // Flag to indicate if the account is unreachable
	UnreachableSince             time.Time            // Timestamp when the account became unreachable
	RateLimitBackoff             map[string]int       `gorm:"serializer:json"` // Exponential backoff multipliers per endpoint
	LastRateLimitHit             map[string]time.Time `gorm:"serializer:json"` // Last time each endpoint was rate limited
}
type Ban struct { // Define the Ban struct
	gorm.Model
	Account          Account           // The account that has a status history.
	AccountID        uint              // The ID of the account.
	Status           Status            // The status of the ban.
	LogType          string            // Type of log entry ("status_change", "account_added", "cookie_update", "check_disabled", "error")
	Message          string            // Detailed message about the log entry
	PreviousStatus   Status            // Store the previous status for better tracking
	TempBanDuration  string            // Duration of the temporary ban (if applicable)
	AffectedGames    string            // Comma-separated list of affected games
	GameSpecificBans map[string]string `gorm:"serializer:json"`           // Map of game titles to enforcement types
	Timestamp        time.Time         `gorm:"default:CURRENT_TIMESTAMP"` // When this log entry was created
	Initiator        string            // "auto_check" or "manual_check" or "system"
	ErrorDetails     string            // For storing error information when relevant
}
type SuppressedNotification struct { // The suppressed notifications table
	gorm.Model
	UserID           string    `gorm:"index"` // The ID of the user.
	NotificationType string    // The type of notification suppressed.
	Content          string    `gorm:"type:text"` // The content of the suppressed notification.
	Timestamp        time.Time `gorm:"index"`     // The timestamp of the suppressed notification.
}

type ShardInfo struct { // Information about application shards
	gorm.Model
	ShardID       int       `gorm:"index"` // The shard ID
	TotalShards   int       // The total number of shards
	InstanceID    string    `gorm:"uniqueIndex"`      // Unique identifier for this instance
	LastHeartbeat time.Time `gorm:"index"`            // Last time this shard reported as alive
	Status        string    `gorm:"default:'active'"` // Status of this shard: active, inactive
	Stats         string    `gorm:"type:text"`        // JSON encoded stats about this shard
}

type ProxyStats struct { // Statistics about HTTP proxies
	gorm.Model
	ProxyURL            string     `gorm:"uniqueIndex"`      // Masked proxy URL (no credentials)
	Status              string     `gorm:"default:'active'"` // Status: active, suspended
	SuccessCount        int64      // Number of successful requests
	FailureCount        int64      // Number of failed requests
	ConsecutiveFailures int        `gorm:"default:0"` // Current consecutive failures
	LastCheck           time.Time  // Last time this proxy was checked
	LastError           string     // Last error message
	RateLimitedUntil    *time.Time // When rate limiting expires (nil if not rate limited)
	Stats               string     `gorm:"type:text"` // Additional stats as JSON
}
type Analytics struct { // The analytics table
	gorm.Model
	Type            string    `gorm:"index"` // The type of analytics entry.
	UserID          string    `gorm:"index"` // The ID of the user.
	GuildID         string    `gorm:"index"` // The ID of the guild.
	CommandName     string    `gorm:"index"` // The name of the command.
	AccountID       uint      `gorm:"index"` // The ID of the account.
	Status          string    `gorm:"index"` // The status of the account.
	PreviousStatus  string    `gorm:"index"` // The previous status of the account.
	Success         bool      // Whether the action was successful.
	ResponseTimeMs  int64     // Response time in milliseconds.
	CaptchaProvider string    // The name of the captcha provider used.
	CaptchaCost     float64   // Cost of the captcha
	ErrorDetails    string    // For storing error information when relevant
	Timestamp       time.Time `gorm:"index"` // When this log entry was created
	Day             string    `gorm:"index"` // YYYY-MM-DD format for easy querying
	ShardID         int       `gorm:"index"` // The shard ID that processed this event
	InstanceID      string    `gorm:"index"` // The instance ID that processed this event
}
type Status string // The status of the account.

const (
	StatusGood          Status = "Good"           // The account status returned as good standing.
	StatusPermaban      Status = "Permaban"       // The account status returned as permanent ban.
	StatusShadowban     Status = "Shadowban"      // The account status returned as shadowban.
	StatusUnknown       Status = "Unknown"        // The account status not known.
	StatusInvalidCookie Status = "Invalid_Cookie" // The account has an invalid SSO cookie.
	StatusTempban       Status = "Temporary"      // The account status returned as temporarily banned.
	StatusRankLocked    Status = "RankLocked"     // The account is restricted from ranked play.
	StatusPartialBan    Status = "PartialBan"     // The account has game-specific bans.
)

type CaptchaProvider string // The type of captcha provider used.

const (
	Capsolver  CaptchaProvider = "capsolver" // The captcha provider is CapSolver.
	EZCaptcha  CaptchaProvider = "ezcaptcha" // The captcha provider is EZCaptcha.
	TwoCaptcha CaptchaProvider = "2captcha"  // The captcha provider is 2Captcha.
)

func (u *UserSettings) EnsureMapsInitialized() {
	if u.NotificationTimes == nil {
		u.NotificationTimes = make(map[string]time.Time)
	}
	if u.ActionCounts == nil {
		u.ActionCounts = make(map[string]int)
	}
	if u.LastActionTimes == nil {
		u.LastActionTimes = make(map[string]time.Time)
	}
	if u.LastCommandTimes == nil {
		u.LastCommandTimes = make(map[string]time.Time)
	}
	if u.RateLimitExpiration == nil {
		u.RateLimitExpiration = make(map[string]time.Time)
	}
	if u.RateLimitBackoff == nil {
		u.RateLimitBackoff = make(map[string]int)
	}
	if u.LastRateLimitHit == nil {
		u.LastRateLimitHit = make(map[string]time.Time)
	}
}

func (u *UserSettings) BeforeCreate(tx *gorm.DB) error {
	u.EnsureMapsInitialized()

	if u.InstallationTime.IsZero() {
		u.InstallationTime = time.Now()
	}

	return nil
}

func (u *UserSettings) AfterFind(tx *gorm.DB) error {
	u.EnsureMapsInitialized()
	return nil
}

func (u *UserSettings) UpdateInteractionContext(guildID string) {
	now := time.Now()

	if u.InstallationType == "" {
		if guildID != "" {
			u.InstallationType = "server"
			u.InstallationGuildID = guildID
		} else {
			u.InstallationType = "direct"
		}
	}

	if guildID != "" {
		u.LastGuildInteraction = now
	} else {
		u.LastDirectInteraction = now
	}

	if u.LastGuildInteraction.After(u.LastDirectInteraction) {
		u.PrimaryInteractionContext = "server"
	} else {
		u.PrimaryInteractionContext = "direct"
	}
}

func (b *Ban) BeforeCreate(tx *gorm.DB) error {
	if b.Timestamp.IsZero() || b.Timestamp.Year() < 1970 {
		b.Timestamp = time.Now()
	}
	return nil
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()

	if a.Created <= 0 {
		a.Created = now.Unix()
	}
	if a.LastCheck <= 0 {
		a.LastCheck = now.Unix()
	}
	if a.LastStatusChange <= 0 {
		a.LastStatusChange = now.Unix()
	}
	if a.GameSpecificBans == nil {
		a.GameSpecificBans = make(map[string]string)
	}

	return nil
}

func (a *Account) BeforeSave(tx *gorm.DB) error {
	now := time.Now()

	if a.Created <= 0 {
		a.Created = now.Unix()
	}

	if a.Created > now.Unix()+86400 {
		a.Created = now.Unix()
	}

	if a.LastCheck > now.Unix()+86400 {
		a.LastCheck = now.Unix()
	}

	return nil
}

func (a *Account) AfterFind(tx *gorm.DB) error {
	if a.GameSpecificBans == nil {
		a.GameSpecificBans = make(map[string]string)
	}
	return nil
}
