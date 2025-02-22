package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

const (
	defaultCooldown         = 1 * time.Hour
	maxNotificationsPerHour = 4
	maxNotificationsPerDay  = 10
	minNotificationInterval = 5 * time.Minute
)

var (
	userNotificationMutex      sync.Mutex
	userNotificationTimestamps = make(map[string]map[string]time.Time)
	adminNotificationCache     = cache.New(5*time.Minute, 10*time.Minute)
	globalLimiter              = NewNotificationLimiter()
	notificationConfigs        = map[string]NotificationConfig{
		"status_change":        {Type: "status_change", Cooldown: 30 * time.Minute, AllowConsolidated: true, MaxPerHour: 10},
		"daily_update":         {Type: "daily_update", Cooldown: 12 * time.Hour, AllowConsolidated: true, MaxPerHour: 3},
		"invalid_cookie":       {Type: "invalid_cookie", Cooldown: 3 * time.Hour, AllowConsolidated: true, MaxPerHour: 4},
		"cookie_expiring_soon": {Type: "cookie_expiring_soon", Cooldown: 12 * time.Hour, AllowConsolidated: true, MaxPerHour: 2},
		"error":                {Type: "error", Cooldown: 30 * time.Minute, AllowConsolidated: true, MaxPerHour: 6},
		"account_added":        {Type: "account_added", Cooldown: 30 * time.Minute, AllowConsolidated: true, MaxPerHour: 8},
		"channel_change":       {Type: "channel_change", Cooldown: 30 * time.Minute, AllowConsolidated: true, MaxPerHour: 6},
		"permaban":             {Type: "permaban", Cooldown: 12 * time.Hour, AllowConsolidated: true, MaxPerHour: 4},
		"shadowban":            {Type: "shadowban", Cooldown: 6 * time.Hour, AllowConsolidated: true, MaxPerHour: 5},
		"temp_ban_update":      {Type: "temp_ban_update", Cooldown: 30 * time.Minute, AllowConsolidated: true, MaxPerHour: 8},
	}
)

type NotificationLimiter struct {
	sync.RWMutex
	userCounts map[string]*NotificationState
	startTime  time.Time
}

func NewNotificationLimiter() *NotificationLimiter {
	return &NotificationLimiter{
		userCounts: make(map[string]*NotificationState),
		startTime:  time.Now(),
	}
}

type NotificationConfig struct {
	Type              string
	Cooldown          time.Duration
	AllowConsolidated bool
	MaxPerHour        int
}

type NotificationState struct {
	hourlyCount int
	dailyCount  int
	lastReset   time.Time
	lastSent    time.Time
}

func NotifyAdmin(s *discordgo.Session, message string) {
	cfg := configuration.Get()
	adminID := cfg.Discord.DeveloperID
	if adminID == "" {
		logger.Log.Error("Developer ID not configured")
		return
	}

	channel, err := s.UserChannelCreate(adminID)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to create DM channel with admin")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Admin Notification",
		Description: message,
		Color:       0xFF0000,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send admin notification")
	}
}

func GetCooldownDuration(userSettings models.UserSettings, notificationType string, defaultCooldown time.Duration) time.Duration {
	cfg := configuration.Get()
	switch notificationType {
	case "status_change":
		return time.Duration(userSettings.StatusChangeCooldown) * time.Hour
	case "daily_update":
		return time.Duration(cfg.Intervals.Notification) * time.Hour
	case "invalid_cookie", "cookie_expiring_soon":
		return time.Duration(cfg.Intervals.CookieExpiration) * time.Hour
	default:
		if config, exists := notificationConfigs[notificationType]; exists {
			return config.Cooldown
		}
		return defaultCooldown
	}
}

/*
func IsDonationsEnabled() bool {
	cfg := configuration.Get()
	return cfg.Donations.Enabled
}
*/

func GetNotificationChannel(s *discordgo.Session, account models.Account, userSettings models.UserSettings) (string, error) {
	if userSettings.NotificationType == "dm" {
		channel, err := s.UserChannelCreate(account.UserID)
		if err != nil {
			return "", fmt.Errorf("failed to create DM channel: %w", err)
		}
		return channel.ID, nil
	}

	if account.ChannelID == "" {
		return "", fmt.Errorf("no channel ID set for account")
	}

	return account.ChannelID, nil
}

func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
func CheckAndNotifyBalance(s *discordgo.Session, userID string, balance float64) {
	cfg := configuration.Get()
	userSettings, err := GetUserSettings(userID)

	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for balance check: %s", userID)
		return
	}

	if time.Since(userSettings.LastBalanceNotification) < time.Duration(cfg.Intervals.Notification)*time.Hour {
		return
	}

	if !IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		logger.Log.Infof("Skipping balance check for disabled service: %s", userSettings.PreferredCaptchaProvider)
		return
	}

	if balance == 0 {
		apiKey, balance, err := GetUserCaptchaKey(userID)
		if err != nil {
			logger.Log.WithError(err).Error("Failed to get captcha balance")
			return
		}

		if apiKey == cfg.CaptchaService.EZCaptcha.ClientKey {
			if balance < cfg.CaptchaService.EZCaptcha.BalanceMin {
				embed := &discordgo.MessageEmbed{
					Title: "Default API Key Balance Low",
					Description: fmt.Sprintf("The bot's default API key balance is currently low (%.2f points). "+
						"To ensure uninterrupted service, consider the following options:", balance),
					Color:     0xFFA500,
					Fields:    BalanceWarningFields(cfg.Donations),
					Timestamp: time.Now().Format(time.RFC3339),
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Thank you for using COD Status Bot!",
					},
				}

				var account models.Account
				if err := database.DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to get account for balance notification")
					return
				}

				if err := SendNotification(s, account, embed, "", "default_key_balance"); err != nil {
					logger.Log.WithError(err).Error("Failed to send default key balance notification")
				}

				NotifyAdminWithCooldown(s, fmt.Sprintf("Default API key balance is low: %.2f", balance), time.Hour*6)
			}
			return
		}
	}

	threshold := getBalanceThreshold(userSettings.PreferredCaptchaProvider)

	if balance < threshold {
		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("Low %s Balance Alert", userSettings.PreferredCaptchaProvider),
			Description: fmt.Sprintf("Your %s balance is currently %.2f points, which is below the recommended threshold of %.2f points.",
				userSettings.PreferredCaptchaProvider, balance, threshold),
			Color: 0xFFA500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "Action Required",
					Value: fmt.Sprintf("Please recharge your %s balance to ensure uninterrupted service for your account checks.",
						userSettings.PreferredCaptchaProvider),
					Inline: false,
				},
				{
					Name:   "Current Provider",
					Value:  userSettings.PreferredCaptchaProvider,
					Inline: true,
				},
				{
					Name:   "Current Balance",
					Value:  fmt.Sprintf("%.2f", balance),
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		var account models.Account
		if err := database.DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to get account for balance notification: %s", userID)
			return
		}

		if err := SendNotification(s, account, embed, "", "balance_warning"); err != nil {
			logger.Log.WithError(err).Errorf("Failed to send balance notification to user %s", userID)
			return
		}

		now := time.Now()
		userSettings.LastBalanceNotification = now
		userSettings.CaptchaBalance = balance
		userSettings.LastBalanceCheck = now

		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastBalanceNotification for user %s", userID)
		}
	}
}

func BalanceWarningFields(config configuration.DonationsConfig) []*discordgo.MessageEmbedField {
	cfg := configuration.Get()
	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Option 1: Add Your Own API Key",
			Value: "Use `/setcaptchaservice` to configure your own Captcha API key. " +
				"This gives you unlimited checks and customizable intervals.",
			Inline: false,
		},
		{
			Name: "Option 2: Wait for Balance Refresh",
			Value: "Continue using the bot's default key with standard rate limits. " +
				"Checks will still work but may be delayed.",
			Inline: false,
		},
	}

	if cfg.Donations.Enabled {
		supportField := &discordgo.MessageEmbedField{
			Name:   "Support the Bot",
			Value:  "Help maintain the service by contributing:\n",
			Inline: false,
		}

		if cfg.Donations.BitcoinAddress != "" {
			supportField.Value += fmt.Sprintf("• Bitcoin: `%s`\n", cfg.Donations.BitcoinAddress)
		}
		if cfg.Donations.CashAppID != "" {
			supportField.Value += fmt.Sprintf("• CashApp: `%s`", cfg.Donations.CashAppID)
		}

		fields = append(fields, supportField)
	}

	return fields
}

func ScheduleBalanceChecks(s *discordgo.Session) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		var users []models.UserSettings
		if err := database.DB.Find(&users).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to fetch users for balance check")
			continue
		}

		for _, user := range users {
			if !IsServiceEnabled(user.PreferredCaptchaProvider) {
				continue
			}

			if user.EZCaptchaAPIKey == "" && user.TwoCaptchaAPIKey == "" && user.CapSolverAPIKey == "" {
				continue
			}

			var apiKey string
			var provider string
			switch {
			case user.PreferredCaptchaProvider == "capsolver" && user.CapSolverAPIKey != "":
				apiKey = user.CapSolverAPIKey
				provider = "capsolver"
			case user.PreferredCaptchaProvider == "2captcha" && user.TwoCaptchaAPIKey != "":
				apiKey = user.TwoCaptchaAPIKey
				provider = "2captcha"
			case user.PreferredCaptchaProvider == "ezcaptcha" && user.EZCaptchaAPIKey != "":
				apiKey = user.EZCaptchaAPIKey
				provider = "ezcaptcha"
			default:
				continue
			}

			isValid, balance, err := ValidateCaptchaKey(apiKey, provider)
			if err != nil {
				logger.Log.WithError(err).Errorf("Failed to validate %s key for user %s", provider, user.UserID)
				continue
			}

			if !isValid {
				if err := DisableUserCaptcha(s, user.UserID, fmt.Sprintf("Invalid %s API key", provider)); err != nil {
					logger.Log.WithError(err).Errorf("Failed to disable captcha for user %s", user.UserID)
				}
				continue
			}

			user.CaptchaBalance = balance
			user.LastBalanceCheck = time.Now()
			if err := database.DB.Save(&user).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to update balance for user %s", user.UserID)
				continue
			}

			CheckAndNotifyBalance(s, user.UserID, balance)
		}
	}
}

func DisableUserCaptcha(s *discordgo.Session, userID string, reason string) error {
	var settings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return err
	}

	settings.TwoCaptchaAPIKey = ""
	switch {
	case IsServiceEnabled("capsolver"):
		settings.PreferredCaptchaProvider = "capsolver"
	case IsServiceEnabled("ezcaptcha"):
		settings.PreferredCaptchaProvider = "ezcaptcha"
	case IsServiceEnabled("2captcha"):
		settings.PreferredCaptchaProvider = "2captcha"
	default:
		settings.PreferredCaptchaProvider = "capsolver"
	}

	settings.EZCaptchaAPIKey = ""
	settings.CustomSettings = false
	settings.CheckInterval = defaultSettings.CheckInterval
	settings.NotificationInterval = defaultSettings.NotificationInterval

	if err := database.DB.Save(&settings).Error; err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title: "Captcha Service Configuration Update",
		Description: fmt.Sprintf("Your captcha service configuration has been updated. Reason: %s\n\n"+
			"Current available services: %s\n"+
			"The bot will use default settings for the available service.",
			reason,
			getEnabledServicesString()),
		Color:     0xFF0000,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return err
	}

	return SendNotification(s, account, embed, "", "captcha_disabled")
}

func getEnabledServicesString() string {
	var enabledServices []string
	if IsServiceEnabled("capsolver") {
		enabledServices = append(enabledServices, "Capsolver")
	}

	if IsServiceEnabled("ezcaptcha") {
		enabledServices = append(enabledServices, "EZCaptcha")
	}

	if IsServiceEnabled("2captcha") {
		enabledServices = append(enabledServices, "2Captcha")
	}
	if len(enabledServices) == 0 {
		return "No services currently enabled"
	}
	return strings.Join(enabledServices, ", ")
}

func (nl *NotificationLimiter) CanSendNotification(userID string, notificationType string) bool {
	if notificationType == "daily_update" {
		return true
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return false
	}

	if userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		return true
	}

	config, exists := notificationConfigs[notificationType]
	if !exists {
		return true
	}

	nl.Lock()
	defer nl.Unlock()

	state, exists := nl.userCounts[userID]
	if !exists {
		state = &NotificationState{
			lastReset: time.Now(),
			lastSent:  time.Now(),
		}
		nl.userCounts[userID] = state
	}

	now := time.Now()

	if now.Sub(state.lastReset) >= time.Hour {
		state.hourlyCount = 0
		state.lastReset = now
	}

	if state.hourlyCount >= config.MaxPerHour {
		storeSuppressedNotification(userID, notificationType, nil, "")
		logger.Log.WithFields(logrus.Fields{
			"userID":       userID,
			"currentCount": state.hourlyCount,
		}).Debug("Notification rate limit reached")
		return false
	}

	if now.Sub(state.lastSent) < minNotificationInterval {
		storeSuppressedNotification(userID, notificationType, nil, "")
		return false
	}

	state.hourlyCount++
	state.dailyCount++
	state.lastSent = now

	return true
}

func SendNotification(s *discordgo.Session, account models.Account, embed *discordgo.MessageEmbed, content, notificationType string) error {
	if !globalLimiter.CanSendNotification(account.UserID, notificationType) {
		storeSuppressedNotification(account.UserID, notificationType, embed, content)
		logger.Log.WithFields(logrus.Fields{
			"userID":           account.UserID,
			"accountTitle":     account.Title,
			"notificationType": notificationType,
		}).Debug("Notification suppressed due to rate limiting")
		return nil
	}

	userSettings, err := GetUserSettings(account.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	now := time.Now()
	lastNotification := userSettings.LastCommandTimes[notificationType]
	cooldownDuration := GetCooldownDuration(userSettings, notificationType, defaultCooldown)

	if !lastNotification.IsZero() && now.Sub(lastNotification) < cooldownDuration {
		logger.Log.Infof("Skipping %s notification for user %s (cooldown)", notificationType, account.UserID)
		return nil
	}

	channelID, err := GetNotificationChannel(s, account, userSettings)
	if err != nil {
		if userSettings.NotificationType == "dm" {
			channel, dmErr := s.UserChannelCreate(account.UserID)
			if dmErr != nil {
				return fmt.Errorf("failed to create DM channel: %w", dmErr)
			}
			channelID = channel.ID
		} else {
			return fmt.Errorf("failed to get notification channel: %w", err)
		}
	}

	_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embed:   embed,
		Content: content,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	userSettings.LastCommandTimes[notificationType] = now
	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update notification timestamp")
	}

	account.LastNotification = now.Unix()
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update account last notification")
	}

	return nil
}

func storeSuppressedNotification(userID, notificationType string, embed *discordgo.MessageEmbed, content string) {
	userNotificationMutex.Lock()
	defer userNotificationMutex.Unlock()

	if userNotificationTimestamps[userID] == nil {
		userNotificationTimestamps[userID] = make(map[string]time.Time)
	}
	userNotificationTimestamps[userID][notificationType] = time.Now()

	if err := database.DB.Create(&models.SuppressedNotification{
		UserID:           userID,
		NotificationType: notificationType,
		Content:          content,
		Timestamp:        time.Now(),
	}).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to store suppressed notification")
	}
}

func NotifyAdminWithCooldown(s *discordgo.Session, message string, cooldownDuration time.Duration) {
	cacheKey := "admin_" + message
	if _, found := adminNotificationCache.Get(cacheKey); found {
		return
	}

	cfg := configuration.Get()
	var admin models.UserSettings
	if err := database.DB.Where("user_id = ?", cfg.Discord.DeveloperID).FirstOrCreate(&admin).Error; err != nil {
		logger.Log.WithError(err).Error("Error getting admin settings")
		return
	}

	now := time.Now()
	notificationType := "admin_" + strings.Split(message, " ")[0]
	lastNotification := admin.LastCommandTimes[notificationType]

	if lastNotification.IsZero() || now.Sub(lastNotification) >= cooldownDuration {
		NotifyAdmin(s, message)
		admin.LastCommandTimes[notificationType] = now
		if err := database.DB.Save(&admin).Error; err != nil {
			logger.Log.WithError(err).Error("Error saving admin settings")
		}
		adminNotificationCache.Set(cacheKey, true, cooldownDuration)
	}
}

func SendGlobalAnnouncement(s *discordgo.Session, userID string) error {
	var userSettings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings for global announcement")
		return result.Error
	}

	if !userSettings.HasSeenAnnouncement {
		channelID, err := getChannelForAnnouncement(s, userID, userSettings)
		if err != nil {
			logger.Log.WithError(err).Error("Error finding recent channel for user")
			return err
		}

		announcementEmbed := CreateAnnouncementEmbed()

		_, err = s.ChannelMessageSendEmbed(channelID, announcementEmbed)
		if err != nil {
			logger.Log.WithError(err).Error("Error sending global announcement")
			return err
		}

		userSettings.HasSeenAnnouncement = true
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Error("Error updating user settings after sending global announcement")
			return err
		}
	}

	return nil
}

func SendAnnouncementToAllUsers(s *discordgo.Session) error {
	var users []models.UserSettings
	if err := database.DB.Find(&users).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching all users")
		return err
	}

	for _, user := range users {
		if err := SendGlobalAnnouncement(s, user.UserID); err != nil {
			logger.Log.WithError(err).Errorf("Failed to send announcement to user %s", user.UserID)
		}
	}

	return nil
}

func NotifyUserAboutDisabledAccount(s *discordgo.Session, account models.Account, reason string) {
	embed := &discordgo.MessageEmbed{
		Title: "Account Disabled",
		Description: fmt.Sprintf("Your account '%s' has been disabled. Reason: %s\n\n"+
			"To re-enable monitoring, please address the issue and use the /togglecheck command to re-enable your account.", account.Title, reason),
		Color:     0xFF0000,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendNotification(s, account, embed, "", "account_disabled")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send account disabled notification to user %s", account.UserID)
	}
}

func NotifyCookieExpiringSoon(s *discordgo.Session, accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	var embedFields []*discordgo.MessageEmbedField
	for _, account := range accounts {
		timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
		if err != nil {
			logger.Log.WithError(err).Errorf("Error checking SSO cookie expiration for account %s", account.Title)
			continue
		}

		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   account.Title,
			Value:  fmt.Sprintf("Cookie expires in %s", FormatDuration(timeUntilExpiration)),
			Inline: false,
		})
	}

	if len(embedFields) == 0 {
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title:       "SSO Cookie Expiration Warning",
		Description: "The following accounts have SSO cookies that will expire soon:",
		Color:       0xFFA500,
		Fields:      embedFields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	return SendNotification(s, accounts[0], embed, "", "cookie_expiring_soon")
}

func SendConsolidatedDailyUpdate(s *discordgo.Session, userID string, userSettings models.UserSettings, accounts []models.Account) {
	if len(accounts) == 0 {
		return
	}

	cfg := configuration.Get()
	notificationInterval := time.Duration(userSettings.NotificationInterval) * time.Hour
	if notificationInterval == 0 {
		notificationInterval = time.Duration(cfg.Intervals.Notification) * time.Hour
	}

	accountsByStatus := make(map[models.Status][]models.Account)
	for _, account := range accounts {
		if !account.IsCheckDisabled && !account.IsExpiredCookie {
			accountsByStatus[account.LastStatus] = append(accountsByStatus[account.LastStatus], account)
		}
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return
	}

	var embedFields []*discordgo.MessageEmbedField
	embedFields = append(embedFields, &discordgo.MessageEmbedField{
		Name: "Summary",
		Value: fmt.Sprintf("Total Accounts: %d\nGood Standing: %d\nBanned: %d\nUnder Review: %d",
			len(accounts),
			len(accountsByStatus[models.StatusGood]),
			len(accountsByStatus[models.StatusPermaban])+len(accountsByStatus[models.StatusTempban]),
			len(accountsByStatus[models.StatusShadowban])),
		Inline: false,
	})

	for status, statusAccounts := range accountsByStatus {
		var description strings.Builder
		for _, account := range statusAccounts {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err != nil {
				logger.Log.WithError(err).Errorf("Error checking expiration for account %s", account.Title)
				continue
			}

			statusSymbol := GetStatusIcon(status)
			description.WriteString(fmt.Sprintf("%s %s: %s\n", statusSymbol, account.Title,
				formatAccountStatus(account, status, timeUntilExpiration)))
		}

		if description.Len() > 0 {
			embedFields = append(embedFields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s Accounts", strings.Title(string(status))),
				Value:  description.String(),
				Inline: false,
			})
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%.2f Hour Update - Account Status Report", userSettings.NotificationInterval),
		Description: "Here's a consolidated update on your monitored accounts:",
		Color:       0x00ff00,
		Fields:      embedFields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /checknow to check any account immediately",
		},
	}

	if err := SendNotification(s, accounts[0], embed, "", "daily_update"); err != nil {
		logger.Log.WithError(err).Errorf("Failed to send consolidated daily update for user %s", userID)
		return
	}

	userSettings.LastDailyUpdateNotification = time.Now()
	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update LastDailyUpdateNotification")
	}

	checkAccountsNeedingAttention(s, accounts, userSettings)
}

func GetStatusIcon(status models.Status) string {
	cfg := configuration.Get()
	switch status {
	case models.StatusGood:
		return cfg.Emojis.CheckCircle
	case models.StatusPermaban:
		return cfg.Emojis.BanCircle
	case models.StatusShadowban:
		return cfg.Emojis.InfoCircle
	case models.StatusTempban:
		return cfg.Emojis.StopWatch
	default:
		return cfg.Emojis.QuestionCircle
	}
}

func formatAccountStatus(account models.Account, status models.Status, timeUntilExpiration time.Duration) string {
	var statusDesc strings.Builder

	switch status {
	case models.StatusGood:
		statusDesc.WriteString(fmt.Sprintf("Good standing | Expires in %s", FormatDuration(timeUntilExpiration)))
	case models.StatusPermaban:
		statusDesc.WriteString("Permanently banned")
	case models.StatusShadowban:
		statusDesc.WriteString("Under review")
	case models.StatusTempban:
		var latestBan models.Ban
		if err := database.DB.Where("account_id = ?", account.ID).
			Order("created_at DESC").
			First(&latestBan).Error; err == nil {
			statusDesc.WriteString(fmt.Sprintf("Temporarily banned (%s remaining)", latestBan.TempBanDuration))
		} else {
			statusDesc.WriteString("Temporarily banned (duration unknown)")
		}
	default:
		statusDesc.WriteString("Unknown status")
	}

	if isVIP, err := CheckVIPStatus(account.SSOCookie); err == nil {
		statusDesc.WriteString(fmt.Sprintf(" | %s", formatVIPStatus(isVIP)))
	}

	statusDesc.WriteString(fmt.Sprintf(" | Checks: %s", formatCheckStatus(account.IsCheckDisabled)))

	return statusDesc.String()
}

func formatVIPStatus(isVIP bool) string {
	if isVIP {
		return "VIP Account"
	}
	return "Regular Account"
}

func formatCheckStatus(isDisabled bool) string {
	if isDisabled {
		return "DISABLED"
	}
	return "ENABLED"
}

func getNotificationType(status models.Status) string {
	switch status {
	case models.StatusPermaban:
		return "permaban"
	case models.StatusShadowban:
		return "shadowban"
	case models.StatusTempban:
		return "tempban"
	default:
		return "status_change"
	}
}

func checkAccountsNeedingAttention(s *discordgo.Session, accounts []models.Account, userSettings models.UserSettings) {
	var expiringAccounts []models.Account
	var errorAccounts []models.Account

	cfg := configuration.Get()
	for _, account := range accounts {
		if !account.IsExpiredCookie {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err != nil {
				errorAccounts = append(errorAccounts, account)
			} else if timeUntilExpiration <= time.Duration(cfg.Intervals.CookieExpiration)*time.Hour {
				expiringAccounts = append(expiringAccounts, account)
			}
		}

		if account.ConsecutiveErrors >= cfg.CaptchaService.MaxRetries {
			errorAccounts = append(errorAccounts, account)
		}
	}

	if len(expiringAccounts) > 0 && time.Since(userSettings.LastCookieExpirationWarning) >= time.Hour*24 {
		if err := NotifyCookieExpiringSoon(s, expiringAccounts); err != nil {
			logger.Log.WithError(err).Error("Failed to send cookie expiration notifications")
		}
	}

	if len(errorAccounts) > 0 && time.Since(userSettings.LastErrorNotification) >= time.Hour*6 {
		notifyAccountErrors(s, errorAccounts, userSettings)
	}
}

func notifyAccountErrors(s *discordgo.Session, errorAccounts []models.Account, userSettings models.UserSettings) {
	if len(errorAccounts) == 0 {
		return
	}

	cfg := configuration.Get()
	embed := &discordgo.MessageEmbed{
		Title:       "Account Check Errors",
		Description: "The following accounts have encountered errors during status checks:",
		Color:       0xFF0000,
		Fields:      make([]*discordgo.MessageEmbedField, 0),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	for _, account := range errorAccounts {
		var errorDescription string
		if account.IsCheckDisabled {
			errorDescription = fmt.Sprintf("Checks disabled - Reason: %s", account.DisabledReason)
		} else if account.ConsecutiveErrors >= cfg.CaptchaService.MaxRetries {
			errorDescription = fmt.Sprintf("Multiple check failures - Last error time: %s",
				account.LastErrorTime.Format("2006-01-02 15:04:05"))
		} else {
			errorDescription = "Unknown error"
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   account.Title,
			Value:  errorDescription,
			Inline: false,
		})
	}

	err := SendNotification(s, errorAccounts[0], embed, "", "error")
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send account errors notification")
	}

	userSettings.LastErrorNotification = time.Now()
	if err = database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update LastErrorNotification timestamp")
	}
}
