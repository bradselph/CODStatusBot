package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func validateRateLimit(userID, action string, duration time.Duration) bool {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return false
	}

	userSettings.EnsureMapsInitialized()

	hasCustomKey := userSettings.CapSolverAPIKey != "" ||
		userSettings.EZCaptchaAPIKey != "" ||
		userSettings.TwoCaptchaAPIKey != ""

	if hasCustomKey {
		logger.Log.Debugf("User %s has custom API key, bypassing rate limit for action: %s", userID, action)
		return true
	}

	logger.Log.Debugf("User %s using default key, checking rate limit for action: %s", userID, action)

	now := time.Now()
	lastAction := userSettings.LastCommandTimes[action]

	if !lastAction.IsZero() && now.Sub(lastAction) < duration {
		return false
	}

	userSettings.LastCommandTimes[action] = now
	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving rate limit")
		return false
	}

	return true
}

func checkActionRateLimit(userID, action string, duration time.Duration) bool {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return false
	}

	userSettings.EnsureMapsInitialized()

	hasCustomKey := userSettings.CapSolverAPIKey != "" ||
		userSettings.EZCaptchaAPIKey != "" ||
		userSettings.TwoCaptchaAPIKey != ""

	if hasCustomKey {
		logger.Log.Debugf("Premium user %s bypassing action rate limit for: %s", userID, action)
		return true
	}

	logger.Log.Debugf("Regular user %s checking action rate limit for: %s (duration: %v)", userID, action, duration)

	now := time.Now()
	lastAction := userSettings.LastActionTimes[action]
	count := userSettings.ActionCounts[action]

	if now.Sub(lastAction) > duration {
		count = 0
	}

	if count >= getActionLimit(action) {
		return false
	}

	tx := database.DB.Begin()
	userSettings.LastActionTimes[action] = now
	userSettings.ActionCounts[action] = count + 1
	if err := tx.Save(&userSettings).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Error saving user settings")
		return false
	}
	tx.Commit()

	return true
}

func getActionLimit(action string) int {
	switch action {
	case "check_account":
		return 25
	case "notification":
		return 30
	default:
		return 10
	}
}

func processUserAccountsWithStats(s *discordgo.Session, userID string, accounts []models.Account) (int, int) {
	if len(accounts) == 0 {
		return 0, 0
	}

	cfg := configuration.Get()
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return 0, 0
	}

	if err := validateUserCaptchaService(userID, userSettings); err != nil {
		logger.Log.WithError(err).Errorf("Captcha service validation failed for user %s", userID)
		notifyUserOfServiceIssue(s, userID, err)
		if strings.Contains(err.Error(), "insufficient balance") {
			return 0, 0
		}
	}

	notificationInterval := time.Duration(userSettings.NotificationInterval) * time.Hour
	if notificationInterval == 0 {
		notificationInterval = time.Duration(cfg.Intervals.Notification) * time.Hour
	}

	shouldSendDaily := time.Since(userSettings.LastDailyUpdateNotification) >= notificationInterval

	var accountsToUpdate, accountsToNotify, accountsForDailyUpdate []models.Account
	successfulChecks := 0
	failedChecks := 0

	for _, account := range accounts {
		if !account.IsCheckDisabled && !account.IsExpiredCookie {
			accountsForDailyUpdate = append(accountsForDailyUpdate, account)
		}

		if !shouldCheckAccount(account, userSettings) {
			continue
		}

		if !checkActionRateLimit(userID, fmt.Sprintf("check_account_%d", account.ID), time.Hour) {
			logger.Log.Infof("Rate limit reached for account %s (ID: %d, User: %s)", account.Title, account.ID, userID)
			continue
		}

		logger.Log.Infof("Starting check for account: %s (ID: %d, User: %s)", account.Title, account.ID, userID)
		startTime := time.Now()
		result, err := CheckAccount(account.SSOCookie, userID, "")
		responseTime := time.Since(startTime).Milliseconds()
		logger.Log.Infof("Completed check for account: %s (ID: %d, User: %s) - Result: %s, Time: %dms", account.Title, account.ID, userID, result, responseTime)

		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to check account %s: %v", account.Title, err)
			LogAccountCheck(account.ID, userID, false, models.StatusUnknown, responseTime, userSettings.PreferredCaptchaProvider, 0, "Check failed")
			handleCheckError(s, &account, err)
			failedChecks++
			continue
		}

		LogAccountCheck(account.ID, userID, true, result, responseTime, userSettings.PreferredCaptchaProvider, 0, "")
		successfulChecks++
		now := time.Now()
		account.LastCheck = now.Unix()
		account.LastSuccessfulCheck = now
		account.ConsecutiveErrors = 0

		if hasStatusChanged(account, result) {
			account.LastStatus = result
			account.LastStatusChange = now.Unix()
			accountsToNotify = append(accountsToNotify, account)
		}

		accountsToUpdate = append(accountsToUpdate, account)
	}

	if len(accountsToUpdate) > 0 {
		DBMutex.Lock()
		if err := database.DB.Save(&accountsToUpdate).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to batch update accounts")
		}
		DBMutex.Unlock()
	}

	if len(accountsToNotify) > 0 {
		processNotifications(s, accountsToNotify, userSettings)
	}

	if shouldSendDaily && len(accountsForDailyUpdate) > 0 {
		SendConsolidatedDailyUpdate(s, userID, userSettings, accountsForDailyUpdate)
	}

	return successfulChecks, failedChecks
}

func processUserAccounts(s *discordgo.Session, userID string, accounts []models.Account) {
	if len(accounts) == 0 {
		return
	}

	cfg := configuration.Get()
	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return
	}

	if err := validateUserCaptchaService(userID, userSettings); err != nil {
		logger.Log.WithError(err).Errorf("Captcha service validation failed for user %s", userID)
		notifyUserOfServiceIssue(s, userID, err)
		if strings.Contains(err.Error(), "insufficient balance") {
			return
		}
	}

	notificationInterval := time.Duration(userSettings.NotificationInterval) * time.Hour
	if notificationInterval == 0 {
		notificationInterval = time.Duration(cfg.Intervals.Notification) * time.Hour
	}

	shouldSendDaily := time.Since(userSettings.LastDailyUpdateNotification) >= notificationInterval

	var accountsToUpdate, accountsToNotify, accountsForDailyUpdate []models.Account

	for _, account := range accounts {
		if !account.IsCheckDisabled && !account.IsExpiredCookie {
			accountsForDailyUpdate = append(accountsForDailyUpdate, account)
		}

		if !shouldCheckAccount(account, userSettings) {
			continue
		}

		if !checkActionRateLimit(userID, fmt.Sprintf("check_account_%d", account.ID), time.Hour) {
			logger.Log.Infof("Rate limit reached for account %s (ID: %d, User: %s)", account.Title, account.ID, userID)
			continue
		}

		logger.Log.Infof("Starting check for account: %s (ID: %d, User: %s)", account.Title, account.ID, userID)
		startTime := time.Now()
		result, err := CheckAccount(account.SSOCookie, userID, "")
		responseTime := time.Since(startTime).Milliseconds()
		logger.Log.Infof("Completed check for account: %s (ID: %d, User: %s) - Result: %s, Time: %dms", account.Title, account.ID, userID, result, responseTime)

		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to check account %s: %v", account.Title, err)
			LogAccountCheck(account.ID, userID, false, models.StatusUnknown, responseTime, userSettings.PreferredCaptchaProvider, 0, "Check failed")
			handleCheckError(s, &account, err)
			continue
		}

		LogAccountCheck(account.ID, userID, true, result, responseTime, userSettings.PreferredCaptchaProvider, 0, "")
		now := time.Now()
		account.LastCheck = now.Unix()
		account.LastSuccessfulCheck = now
		account.ConsecutiveErrors = 0

		if hasStatusChanged(account, result) {
			account.LastStatus = result
			account.LastStatusChange = now.Unix()
			accountsToNotify = append(accountsToNotify, account)
		}

		accountsToUpdate = append(accountsToUpdate, account)
	}

	if len(accountsToUpdate) > 0 {
		DBMutex.Lock()
		if err := database.DB.Save(&accountsToUpdate).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to batch update accounts")
		}
		DBMutex.Unlock()
	}

	if len(accountsToNotify) > 0 {
		processNotifications(s, accountsToNotify, userSettings)
	}

	if shouldSendDaily && len(accountsForDailyUpdate) > 0 {
		SendConsolidatedDailyUpdate(s, userID, userSettings, accountsForDailyUpdate)
	}
}

/*
	func processUserAccounts(s *discordgo.Session, userID string, accounts []models.Account) {
		logger.Log.Debugf("Processing %d accounts for user %s", len(accounts), userID)

		userSettings, err := GetUserSettings(userID)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to get user settings for %s", userID)
			return
		}

		for _, account := range accounts {
			if account.IsCheckDisabled || account.IsExpiredCookie {
				continue
			}

			if time.Since(time.Unix(account.LastCheck, 0)) < time.Duration(userSettings.CheckInterval)*time.Minute {
				continue
			}

			result, err := CheckAccount(account.SSOCookie, userID, "")
			if err != nil {
				logger.Log.WithError(err).Errorf("Error checking account %s", account.Title)
				account.ConsecutiveErrors++
				account.LastErrorTime = time.Now()

				if account.ConsecutiveErrors >= maxConsecutiveErrors {
					disableAccount(s, account, fmt.Sprintf("Too many consecutive errors: %v", err))
				} else {
					database.DB.Save(&account)
				}
				continue
			}

			account.LastCheck = time.Now().Unix()
			account.ConsecutiveErrors = 0
			account.LastSuccessfulCheck = time.Now()

			HandleStatusChange(s, account, result, userSettings)

			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Error("Failed to save account after check")
			}
		}
	}
*/
func notifyUserOfServiceIssue(s *discordgo.Session, userID string, err error) {
	cfg := configuration.Get()
	if userID != cfg.Discord.DeveloperID {
		return
	}

	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to create DM channel for service issue")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Service Issue Detected",
		Description: fmt.Sprintf("A service issue has been detected: %v\nUser ID: %s", err, userID),
		Color:       0xFF0000,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Issue Type",
				Value:  "Captcha Service Configuration",
				Inline: true,
			},
			{
				Name:   "Timestamp",
				Value:  time.Now().Format(time.RFC3339),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if _, err = s.ChannelMessageSendEmbed(channel.ID, embed); err != nil {
		logger.Log.WithError(err).Error("Failed to send admin service issue notification")
	}
}

func shouldCheckAccount(account models.Account, settings models.UserSettings) bool {
	cfg := configuration.Get()

	if account.IsCheckDisabled {
		logger.Log.Infof("Account %s (ID: %d, User: %s) is disabled, skipping check. Reason: %s", account.Title, account.ID, account.UserID, account.DisabledReason)
		return false
	}

	if account.IsExpiredCookie {
		logger.Log.Infof("Account %s (ID: %d, User: %s) has expired cookie, skipping check", account.Title, account.ID, account.UserID)
		return false
	}

	if account.LastCheck == 0 {
		logger.Log.Infof("Account %s (ID: %d, User: %s) has never been checked, allowing check", account.Title, account.ID, account.UserID)
		return true
	}

	lastCheckTime := time.Unix(account.LastCheck, 0)
	hasCustomKey := settings.CapSolverAPIKey != "" || settings.EZCaptchaAPIKey != "" || settings.TwoCaptchaAPIKey != ""

	var checkInterval time.Duration
	if account.IsPermabanned {
		checkInterval = time.Duration(cfg.Intervals.PermaBanCheck) * time.Hour
		logger.Log.Debugf("Account %s (ID: %d, User: %s) is permabanned, using permaban check interval: %v", account.Title, account.ID, account.UserID, checkInterval)
	} else {
		userInterval := settings.CheckInterval
		if userInterval < 1 {
			userInterval = cfg.Intervals.Check
		}
		checkInterval = time.Duration(userInterval) * time.Minute
		logger.Log.Debugf("Account %s using check interval: %v (user: %d, default: %d)", account.Title, checkInterval, settings.CheckInterval, cfg.Intervals.Check)
	}

	if !hasCustomKey {
		defaultRateLimit := cfg.RateLimits.Default
		if checkInterval < defaultRateLimit {
			checkInterval = defaultRateLimit
			logger.Log.Debugf("Account %s (ID: %d, User: %s) using default rate limit instead: %v (regular user)", account.Title, account.ID, account.UserID, checkInterval)
		}
	} else {
		logger.Log.Debugf("Account %s (ID: %d, User: %s) is premium user, using custom interval: %v", account.Title, account.ID, account.UserID, checkInterval)
	}

	if account.ConsecutiveErrors > cfg.ErrorHandling.MaxConsecutiveErrors && !account.LastErrorTime.IsZero() {
		errorCooldown := time.Duration(cfg.Intervals.Cooldown) * time.Hour
		if time.Since(account.LastErrorTime) < errorCooldown {
			logger.Log.Debugf("Account %s in error cooldown, skipping check", account.Title)
			return false
		}
	}

	timeSinceLastCheck := time.Since(lastCheckTime)
	shouldCheck := timeSinceLastCheck >= checkInterval

	logger.Log.Infof("Account %s (ID: %d, User: %s) check decision: should=%v, timeSince=%v, interval=%v, premium=%v",
		account.Title, account.ID, account.UserID, shouldCheck, timeSinceLastCheck, checkInterval, hasCustomKey)

	return shouldCheck
}

func hasStatusChanged(account models.Account, newStatus models.Status) bool {
	if account.LastStatus == models.StatusUnknown {
		return true
	}
	return account.LastStatus != newStatus
}

func handleCheckError(s *discordgo.Session, account *models.Account, err error) {
	cfg := configuration.Get()
	account.ConsecutiveErrors++
	account.LastErrorTime = time.Now()

	if err := database.DB.Save(account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to update account error status: %s", account.Title)
		return
	}

	if account.ConsecutiveErrors >= cfg.CaptchaService.MaxRetries {
		disableAccount(s, *account, fmt.Sprintf("Max consecutive errors reached (%d). Last error: %v",
			cfg.CaptchaService.MaxRetries, err))
	}
}

func processNotifications(s *discordgo.Session, accounts []models.Account, userSettings models.UserSettings) {
	for _, account := range accounts {
		if !validateRateLimit(account.UserID, "notification", time.Hour) {
			logger.Log.Infof("Notification rate limit reached for user %s", account.UserID)
			continue
		}

		logger.Log.Infof("Processing notification for account %s with status: %s", account.Title, account.LastStatus)
		HandleStatusChange(s, account, account.LastStatus, &userSettings)
	}
}

func isComingFromBannedState(account models.Account) bool {
	var previousBan models.Ban
	result := database.DB.Where("account_id = ?", account.ID).
		Order("timestamp DESC").
		First(&previousBan)

	if result.Error != nil {
		logger.Log.WithError(result.Error).
			Errorf("Could not check previous ban state for account %s", account.Title)
		return false
	}

	bannedStates := []models.Status{
		models.StatusPermaban,
		models.StatusShadowban,
		models.StatusTempban,
	}

	for _, state := range bannedStates {
		if previousBan.Status == state {
			return true
		}
	}
	return false
}

func shouldCheckExpiration(account models.Account) bool {
	cfg := configuration.Get()
	if account.IsExpiredCookie {
		return false
	}

	timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
	if err != nil {
		return false
	}

	return timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cfg.Intervals.CookieExpiration)*time.Hour
}

func validateUserCaptchaService(userID string, userSettings models.UserSettings) error {
	if !IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		return fmt.Errorf("captcha service %s is disabled", userSettings.PreferredCaptchaProvider)
	}

	if userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		_, balance, err := GetUserCaptchaKey(userID)
		if err != nil {
			return fmt.Errorf("failed to validate captcha key: %w", err)
		}
		if balance <= 0 {
			return fmt.Errorf("insufficient captcha balance: %.2f", balance)
		}
	}

	return nil
}

func ValidateDefaultCapsolverConfig() error {
	cfg := configuration.Get()

	if !cfg.CaptchaService.Capsolver.Enabled {
		return fmt.Errorf("capsolver service is disabled")
	}

	if cfg.CaptchaService.Capsolver.ClientKey == "" {
		return fmt.Errorf("capsolver client key not configured")
	}
	if cfg.CaptchaService.Capsolver.AppID == "" {
		return fmt.Errorf("capsolver App ID not configured")
	}

	isValid, _, err := ValidateCaptchaKey(cfg.CaptchaService.Capsolver.ClientKey, "capsolver")
	if err != nil {
		return fmt.Errorf("failed to validate Capsolver key: %w", err)
	}
	if !isValid {
		return fmt.Errorf("capsolver API key is invalid")
	}

	return nil
}

func GetCheckStatus(isCheckDisabled bool) string {
	if isCheckDisabled {
		return "Disabled"
	}
	return "Enabled"
}

func checkUserBalance(s *discordgo.Session, user models.UserSettings) {
	apiKey, balance, err := GetUserCaptchaKey(user.UserID)
	if err != nil || apiKey == "" {
		return
	}

	var threshold float64
	switch user.PreferredCaptchaProvider {
	case "ezcaptcha":
		threshold = 250
	case "2captcha":
		threshold = 0.25
	default:
		threshold = 250
	}

	if balance < threshold && time.Since(user.LastBalanceNotification) >= 24*time.Hour {
		channel, err := s.UserChannelCreate(user.UserID)
		if err != nil {
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Low Balance Warning",
			Description: fmt.Sprintf("Your %s balance is low: %.2f points", user.PreferredCaptchaProvider, balance),
			Color:       0xFFA500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Recommended Minimum",
					Value:  fmt.Sprintf("%.2f points", threshold),
					Inline: true,
				},
				{
					Name:   "Action Required",
					Value:  "Please add funds to continue monitoring",
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		if _, err := s.ChannelMessageSendEmbed(channel.ID, embed); err == nil {
			user.LastBalanceNotification = time.Now()
			database.DB.Save(&user)
		}
	}

	user.LastBalanceCheck = time.Now()
	user.CaptchaBalance = balance
	database.DB.Save(&user)
}

func cleanupErrors() {
	cfg := configuration.Get()
	cutoffTime := time.Now().Add(-time.Duration(cfg.ErrorHandling.ErrorNotificationCooldownHours) * time.Hour)

	var accounts []models.Account
	if err := database.DB.Where("consecutive_errors > ? AND last_error_time < ?", 0, cutoffTime).Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch accounts for error cleanup")
		return
	}

	cleanupCount := 0
	for _, account := range accounts {
		if account.ConsecutiveErrors > 0 && time.Since(account.LastErrorTime) > 24*time.Hour {
			account.ConsecutiveErrors = 0
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to reset error count for account %s", account.Title)
			} else {
				cleanupCount++
			}
		}
	}

	if cleanupCount > 0 {
		logger.Log.Infof("Reset error counts for %d accounts", cleanupCount)
	}
}

func cleanupEdgeCases() {
	var accounts []models.Account
	if err := database.DB.Where("is_check_disabled = ? AND disabled_reason LIKE ?", true, "%consecutive errors%").Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch disabled accounts for edge case cleanup")
		return
	}

	reenableCount := 0
	for _, account := range accounts {
		if time.Since(account.LastErrorTime) > 48*time.Hour {
			account.IsCheckDisabled = false
			account.DisabledReason = ""
			account.ConsecutiveErrors = 0
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to re-enable account %s", account.Title)
			} else {
				reenableCount++
				logger.Log.Infof("Re-enabled account %s after extended cooldown", account.Title)
			}
		}
	}

	if reenableCount > 0 {
		logger.Log.Infof("Re-enabled %d accounts after extended error cooldown", reenableCount)
	}
}
