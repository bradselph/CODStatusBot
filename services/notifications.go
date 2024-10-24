package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/webserver"
	"github.com/bwmarrin/discordgo"
)

func NotifyAdminWithCooldown(s *discordgo.Session, message string, cooldownDuration time.Duration) {
	webserver.NotificationMutex.Lock()
	defer webserver.NotificationMutex.Unlock()

	notificationType := "admin_" + strings.Split(message, " ")[0]

	_, found := adminNotificationCache.Get(notificationType)
	if !found {
		NotifyAdmin(s, message)
		adminNotificationCache.Set(notificationType, time.Now(), cooldownDuration)
	} else {
		logger.Log.Infof("Skipping admin notification '%s' due to cooldown", notificationType)
	}
}

func NotifyAdmin(s *discordgo.Session, message string) {
	adminID := os.Getenv("DEVELOPER_ID")
	if adminID == "" {
		logger.Log.Error("DEVELOPER_ID not set in environment variables")
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

func CheckNotificationCooldown(userID string, notificationType string, cooldownDuration time.Duration) (bool, error) {
	var settings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return false, err
	}

	var lastNotification time.Time
	switch notificationType {
	case "balance":
		lastNotification = settings.LastBalanceNotification
	case "error":
		lastNotification = settings.LastErrorNotification
	case "disabled":
		lastNotification = settings.LastDisabledNotification
	default:
		return false, fmt.Errorf("unknown notification type: %s", notificationType)
	}

	if time.Since(lastNotification) >= cooldownDuration {
		return true, nil
	}
	return false, nil
}

func UpdateNotificationTimestamp(userID string, notificationType string) error {
	var settings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return err
	}

	now := time.Now()
	switch notificationType {
	case "balance":
		settings.LastBalanceNotification = now
	case "error":
		settings.LastErrorNotification = now
	case "disabled":
		settings.LastDisabledNotification = now
	default:
		return fmt.Errorf("unknown notification type: %s", notificationType)
	}

	return database.DB.Save(&settings).Error
}

func SendNotification(s *discordgo.Session, account models.Account, embed *discordgo.MessageEmbed, content, notificationType string) error {
	if account.IsCheckDisabled {
		logger.Log.Infof("Skipping notification for disabled account %s", account.Title)
		return nil
	}

	userSettings, err := GetUserSettings(account.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	config, ok := notificationConfigs[notificationType]
	if !ok {
		return fmt.Errorf("unknown notification type: %s (account: %s, user: %s)", notificationType, account.Title, account.UserID)
	}

	userNotificationMutex.Lock()
	defer userNotificationMutex.Unlock()

	if _, exists := userNotificationTimestamps[account.UserID]; !exists {
		userNotificationTimestamps[account.UserID] = make(map[string]time.Time)
	}

	lastNotification, exists := userNotificationTimestamps[account.UserID][notificationType]
	now := time.Now()

	cooldownDuration := GetCooldownDuration(userSettings, notificationType, config.Cooldown)

	if exists && now.Sub(lastNotification) < cooldownDuration {
		logger.Log.Infof("Skipping %s notification for user %s (cooldown)", notificationType, account.UserID)
		return nil
	}

	channelID, err := GetNotificationChannel(s, account, userSettings)
	if err != nil {
		return fmt.Errorf("failed to get notification channel: %w", err)
	}

	_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embed:   embed,
		Content: content,
	})
	if err != nil {
		if strings.Contains(err.Error(), "Missing Access") || strings.Contains(err.Error(), "Unknown Channel") {
			logger.Log.Warnf("Bot might have been removed from the channel or server for user %s", account.UserID)
			return fmt.Errorf("bot might have been removed: %w", err)
		}
		return fmt.Errorf("failed to send notification: %w", err)
	}

	logger.Log.Infof("%s notification sent to user %s", notificationType, account.UserID)
	userNotificationTimestamps[account.UserID][notificationType] = now

	account.LastNotification = now.Unix()
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to update LastNotification for account %s", account.Title)
	}

	return nil
}

func GetCooldownDuration(userSettings models.UserSettings, notificationType string, defaultCooldown time.Duration) time.Duration {
	switch notificationType {
	case "status_change":
		return time.Duration(userSettings.StatusChangeCooldown) * time.Hour
	case "daily_update", "invalid_cookie", "cookie_expiring_soon":
		return time.Duration(userSettings.NotificationInterval) * time.Hour
	default:
		return defaultCooldown
	}
}

func GetNotificationChannel(s *discordgo.Session, account models.Account, userSettings models.UserSettings) (string, error) {
	if userSettings.NotificationType == "dm" {
		channel, err := s.UserChannelCreate(account.UserID)
		if err != nil {
			return "", fmt.Errorf("failed to create DM channel: %w", err)
		}
		return channel.ID, nil
	}
	return account.ChannelID, nil
}

func SendConsolidatedDailyUpdate(s *discordgo.Session, userID string, userSettings models.UserSettings, accounts []models.Account) {
	if len(accounts) == 0 {
		return
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return
	}

	var embedFields []*discordgo.MessageEmbedField

	for _, account := range accounts {
		var description string
		if account.IsExpiredCookie {
			description = "SSO cookie has expired. Please update using /updateaccount command."
		} else {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err != nil {
				description = "Error checking SSO cookie expiration. Please check manually."
			} else if timeUntilExpiration > 0 {
				description = fmt.Sprintf("Status: %s. Cookie expires in %s.", account.LastStatus, FormatExpirationTime(account.SSOCookieExpiration))
			} else {
				description = "SSO cookie has expired. Please update using /updateaccount command."
			}
		}

		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   account.Title,
			Value:  description,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%.2f Hour Update - Multiple Accounts", userSettings.NotificationInterval),
		Description: "Here's an update on your monitored accounts:",
		Color:       0x00ff00,
		Fields:      embedFields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	err = SendNotification(s, accounts[0], embed, "", "daily_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send consolidated daily update for user %s", userID)
	} else {
		userSettings.LastDailyUpdateNotification = time.Now()
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastDailyUpdateNotification for user %s", userID)
		}
	}
}

func NotifyCookieExpiringSoon(s *discordgo.Session, accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	userID := accounts[0].UserID
	logger.Log.Infof("Sending cookie expiration warning for user %s", userID)

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

	embed := &discordgo.MessageEmbed{
		Title:       "SSO Cookie Expiration Warning",
		Description: "The following accounts have SSO cookies that will expire soon:",
		Color:       0xFFA500,
		Fields:      embedFields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	return SendNotification(s, accounts[0], embed, "", "cookie_expiring_soon")
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

func CheckAndNotifyBalance(s *discordgo.Session, userID string, balance float64) {
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for balance check: %s", userID)
		return
	}

	if time.Since(userSettings.LastBalanceNotification) < 24*time.Hour {
		return
	}

	var thresholds = map[string]float64{
		"ezcaptcha": 1000, // EZCaptcha threshold
		"2captcha":  1.00, // 2captcha threshold (different scale)
	}

	threshold := thresholds[userSettings.PreferredCaptchaProvider]
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

		err := SendNotification(s, account, embed, "", "balance_warning")
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to send balance notification to user %s", userID)
			return
		}

		userSettings.LastBalanceNotification = time.Now()
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastBalanceNotification for user %s", userID)
		}
	}
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
			if user.EZCaptchaAPIKey == "" && user.TwoCaptchaAPIKey == "" {
				continue
			}

			var apiKey string
			var provider string
			if user.PreferredCaptchaProvider == "2captcha" && user.TwoCaptchaAPIKey != "" {
				apiKey = user.TwoCaptchaAPIKey
				provider = "2captcha"
			} else if user.PreferredCaptchaProvider == "ezcaptcha" && user.EZCaptchaAPIKey != "" {
				apiKey = user.EZCaptchaAPIKey
				provider = "ezcaptcha"
			} else {
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

	settings.EZCaptchaAPIKey = ""
	settings.TwoCaptchaAPIKey = ""
	settings.PreferredCaptchaProvider = "ezcaptcha"
	settings.CustomSettings = false
	settings.CheckInterval = defaultSettings.CheckInterval
	settings.NotificationInterval = defaultSettings.NotificationInterval

	if err := database.DB.Save(&settings).Error; err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title: "Captcha Service Disabled",
		Description: fmt.Sprintf("Your captcha service has been disabled. Reason: %s\n"+
			"The bot will now use default settings and the default captcha provider.", reason),
		Color:     0xFF0000,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return err
	}

	return SendNotification(s, account, embed, "", "captcha_disabled")
}

func SendTempBanUpdateNotification(s *discordgo.Session, account models.Account, remainingTime time.Duration) {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Temporary Ban Update", account.Title),
		Description: fmt.Sprintf("Your account is still temporarily banned. Remaining time: %v", remainingTime),
		Color:       GetColorForStatus(models.StatusTempban, false, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	err := SendNotification(s, account, embed, "", "temp_ban_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send temporary ban update for account %s", account.Title)
	}
}
