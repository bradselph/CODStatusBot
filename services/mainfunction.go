package services

import (
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NotificationConfig struct {
	Type              string
	Cooldown          time.Duration
	AllowConsolidated bool
}

const (
	maxConsecutiveErrors          = 5
	balanceNotificationThreshold  = 1000
	maxUserErrorNotifications     = 3
	userErrorNotificationCooldown = 24 * time.Hour
	balanceNotificationInterval   = 24 * time.Hour
)

var (
	checkInterval               float64
	notificationInterval        float64
	cooldownDuration            float64
	sleepDuration               int
	cookieCheckIntervalPermaban float64
	statusChangeCooldown        float64
	globalNotificationCooldown  float64
	cookieExpirationWarning     float64
	tempBanUpdateInterval       float64
	defaultRateLimit            time.Duration
	checkNowRateLimit           time.Duration
	userNotificationMutex       sync.Mutex
	userNotificationTimestamps  = make(map[string]map[string]time.Time)
	DBMutex                     sync.Mutex
	userErrorNotifications      = make(map[string][]time.Time)
	userErrorNotificationMutex  sync.Mutex
	adminNotificationCache      = cache.New(5*time.Minute, 10*time.Minute)
	notificationConfigs         = map[string]NotificationConfig{
		"status_change":        {Cooldown: time.Hour, AllowConsolidated: false},
		"permaban":             {Cooldown: 24 * time.Hour, AllowConsolidated: false},
		"daily_update":         {Cooldown: 0, AllowConsolidated: true},
		"invalid_cookie":       {Cooldown: 6 * time.Hour, AllowConsolidated: true},
		"cookie_expiring_soon": {Cooldown: 24 * time.Hour, AllowConsolidated: true},
		"temp_ban_update":      {Cooldown: time.Hour, AllowConsolidated: false},
		"error":                {Cooldown: time.Hour, AllowConsolidated: false},
		"account_added":        {Cooldown: time.Hour, AllowConsolidated: false},
	}
)

func init() {
	if err := godotenv.Load(); err != nil {
		logger.Log.WithError(err).Error("Failed to load .env file")
	}

	checkInterval = GetEnvFloat("CHECK_INTERVAL", 15)
	notificationInterval = GetEnvFloat("NOTIFICATION_INTERVAL", 24)
	cooldownDuration = GetEnvFloat("COOLDOWN_DURATION", 6)
	sleepDuration = GetEnvInt("SLEEP_DURATION", 1)
	cookieCheckIntervalPermaban = GetEnvFloat("COOKIE_CHECK_INTERVAL_PERMABAN", 24)
	statusChangeCooldown = GetEnvFloat("STATUS_CHANGE_COOLDOWN", 1)
	globalNotificationCooldown = GetEnvFloat("GLOBAL_NOTIFICATION_COOLDOWN", 2)
	cookieExpirationWarning = GetEnvFloat("COOKIE_EXPIRATION_WARNING", 24)
	tempBanUpdateInterval = GetEnvFloat("TEMP_BAN_UPDATE_INTERVAL", 24)
	defaultRateLimit = time.Duration(GetEnvInt("DEFAULT_RATE_LIMIT", 5)) * time.Minute
	checkNowRateLimit = time.Duration(GetEnvInt("CHECK_NOW_RATE_LIMIT", 3600)) * time.Second

	logger.Log.Infof("Loaded config: CHECK_INTERVAL=%.2f, NOTIFICATION_INTERVAL=%.2f, COOLDOWN_DURATION=%.2f, SLEEP_DURATION=%d, COOKIE_CHECK_INTERVAL_PERMABAN=%.2f, STATUS_CHANGE_COOLDOWN=%.2f, GLOBAL_NOTIFICATION_COOLDOWN=%.2f, COOKIE_EXPIRATION_WARNING=%.2f, TEMP_BAN_UPDATE_INTERVAL=%.2f, CHECK_NOW_RATE_LIMIT=%v, DEFAULT_RATE_LIMIT=%v",
		checkInterval, notificationInterval, cooldownDuration, sleepDuration, cookieCheckIntervalPermaban, statusChangeCooldown, globalNotificationCooldown, cookieExpirationWarning, tempBanUpdateInterval, checkNowRateLimit, defaultRateLimit)
}

func GetEnvFloat(key string, fallback float64) float64 {
	value := GetEnvFloatRaw(key, fallback)
	// Convert hours to minutes for certain settings
	if key == "CHECK_INTERVAL" || key == "SLEEP_DURATION" || key == "DEFAULT_RATE_LIMIT" {
		return value
	}
	// All other values are in hours, so we don't need to convert them
	return value
}

func GetEnvFloatRaw(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return floatValue
		}
		logger.Log.WithError(err).Errorf("Failed to parse %s, using fallback value", key)
	}
	return fallback
}

func GetEnvInt(key string, fallback int) int {
	return GetEnvIntRaw(key, fallback)
}

func GetEnvIntRaw(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.Atoi(value)
		if err == nil {
			return intValue
		}
		logger.Log.WithError(err).Errorf("Failed to parse %s, using fallback value", key)
	}
	return fallback
}

func getStatusDescription(status models.Status, accountTitle string, ban models.Ban) string {
	affectedGames := strings.Split(ban.AffectedGames, ",")
	gamesList := strings.Join(affectedGames, ", ")

	switch status {
	case models.StatusPermaban:
		return fmt.Sprintf("The account %s has been permanently banned.\nAffected games: %s", accountTitle, gamesList)
	case models.StatusShadowban:
		return fmt.Sprintf("The account %s is currently shadowbanned.\nAffected games: %s", accountTitle, gamesList)
	case models.StatusTempban:
		return fmt.Sprintf("The account %s is temporarily banned for %s.\nAffected games: %s", accountTitle, ban.TempBanDuration, gamesList)
	default:
		return fmt.Sprintf("The account %s is currently not banned.", accountTitle)
	}
}
func createTempBanLiftedEmbed(account models.Account) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Temporary Ban Lifted", account.Title),
		Description: fmt.Sprintf("The temporary ban for account %s has been lifted. The account is now in good standing.", account.Title),
		Color:       GetColorForStatus(models.StatusGood, false, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func createTempBanEscalatedEmbed(account models.Account) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Temporary Ban Escalated", account.Title),
		Description: fmt.Sprintf("The temporary ban for account %s has been escalated to a permanent ban.", account.Title),
		Color:       GetColorForStatus(models.StatusPermaban, false, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func createTempBanStillActiveEmbed(account models.Account, status models.Status) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Temporary Ban Update", account.Title),
		Description: fmt.Sprintf("The temporary ban for account %s is still in effect. Current status: %s", account.Title, status),
		Color:       GetColorForStatus(status, false, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func checkAccountAfterTempBan(s *discordgo.Session, account models.Account) {
	result, err := CheckAccount(account.SSOCookie, account.UserID, "")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to check account %s after temporary ban duration", account.Title)
		return
	}

	var embed *discordgo.MessageEmbed
	switch result {
	case models.StatusGood:
		embed = createTempBanLiftedEmbed(account)
	case models.StatusPermaban:
		embed = createTempBanEscalatedEmbed(account)
	default:
		embed = createTempBanStillActiveEmbed(account, result)
	}

	err = SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), "temp_ban_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send temporary ban update message for account %s", account.Title)
	}
}

func createStatusChangeEmbed(account models.Account, newStatus models.Status, ban models.Ban) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - %s", account.Title, EmbedTitleFromStatus(newStatus)),
		Description: getStatusDescription(newStatus, account.Title, ban),
		Color:       GetColorForStatus(newStatus, account.IsExpiredCookie, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func EmbedTitleFromStatus(status models.Status) string {
	switch status {
	case models.StatusTempban:
		return "TEMPORARY BAN DETECTED"
	case models.StatusPermaban:
		return "PERMANENT BAN DETECTED"
	case models.StatusShadowban:
		return "SHADOWBAN DETECTED"
	default:
		return "ACCOUNT NOT BANNED"
	}
}

func sendIndividualDailyUpdate(s *discordgo.Session, account models.Account, userSettings models.UserSettings, GetColorForStatus func(models.Status, bool, bool) int) {
	now := time.Now()
	if now.Sub(userSettings.LastDailyUpdateNotification) < 24*time.Hour {
		logger.Log.Infof("Skipping daily update for account %s as last notification was less than 24 hours ago", account.Title)
		return
	}

	logger.Log.Infof("Sending individual daily update for account %s", account.Title)

	var description string
	if account.IsExpiredCookie {
		description = fmt.Sprintf("The SSO cookie for account %s has expired. Please update the cookie using the /updateaccount command or delete the account using the /removeaccount command.", account.Title)
	} else {
		timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
		if err != nil {
			logger.Log.WithError(err).Errorf("Error checking SSO cookie expiration for account %s", account.Title)
			description = fmt.Sprintf("An error occurred while checking the SSO cookie expiration for account %s. Please check the account status manually.", account.Title)
		} else if timeUntilExpiration > 0 {
			description = fmt.Sprintf("The last status of account %s was %s. SSO cookie will expire in %s.", account.Title, account.LastStatus, FormatExpirationTime(account.SSOCookieExpiration))
		} else {
			description = fmt.Sprintf("The SSO cookie for account %s has expired. Please update the cookie using the /updateaccount command or delete the account using the /removeaccount command.", account.Title)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%.2f Hour Update - %s", notificationInterval, account.Title),
		Description: description,
		Color:       GetColorForStatus(account.LastStatus, account.IsExpiredCookie, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	err := SendNotification(s, account, embed, "", "daily_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send individual daily update message for account %s", account.Title)
	} else {
		userSettings.LastDailyUpdateNotification = now
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastDailyUpdateNotification for user %s", account.UserID)
		}
	}

	account.Last24HourNotification = now
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to update Last24HourNotification for account %s", account.Title)
	}
}

func HandleStatusChange(s *discordgo.Session, account models.Account, newStatus models.Status, userSettings models.UserSettings) {
	DBMutex.Lock()
	defer DBMutex.Unlock()

	now := time.Now()

	// Check cooldown
	if now.Sub(userSettings.LastStatusChangeNotification) < time.Duration(userSettings.StatusChangeCooldown)*time.Hour {
		logger.Log.Infof("Skipping status change notification for account %s (cooldown)", account.Title)
		return
	}

	//previousStatus := account.LastStatus
	account.LastStatus = newStatus
	account.LastStatusChange = now.Unix()
	account.IsPermabanned = newStatus == models.StatusPermaban
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to save account changes for account %s", account.Title)
		return
	}

	logger.Log.Infof("Account %s status changed to %s", account.Title, newStatus)

	ban := models.Ban{
		Account:   account,
		Status:    newStatus,
		AccountID: account.ID,
	}

	if newStatus == models.StatusTempban {
		//banDuration := 7 * 24 * time.Hour

		ban.TempBanDuration = calculateBanDuration(now)
	}

	if err := database.DB.Create(&ban).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to create new ban record for account %s", account.Title)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - %s", account.Title, EmbedTitleFromStatus(newStatus)),
		Description: getStatusDescription(newStatus, account.Title, ban),
		Color:       GetColorForStatus(newStatus, account.IsExpiredCookie, account.IsCheckDisabled),
		Timestamp:   now.Format(time.RFC3339),
	}

	notificationType := "status_change"
	if newStatus == models.StatusPermaban {
		notificationType = "permaban"
	}

	err := SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), notificationType)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send status update message for account %s", account.Title)
	} else {
		// Update last notification time
		userSettings.LastStatusChangeNotification = now
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastStatusChangeNotification for user %s", account.UserID)
		}
	}

	// Schedule temp ban notifications for non-permabanned accounts
	if newStatus == models.StatusTempban {
		go ScheduleTempBanNotification(s, account, ban.TempBanDuration)
	}
}

func CheckAccounts(s *discordgo.Session) {
	logger.Log.Info("Starting periodic account check")
	var accounts []models.Account
	if err := database.DB.Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to fetch accounts from the database")
		return
	}

	accountsByUser := make(map[string][]models.Account)
	for _, account := range accounts {
		accountsByUser[account.UserID] = append(accountsByUser[account.UserID], account)
	}

	for userID, userAccounts := range accountsByUser {
		go func(uid string, accounts []models.Account) {
			userSettings, err := GetUserSettings(uid)
			if err != nil {
				logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", uid)
				return
			}

			var accountsToUpdate []models.Account
			var dailyUpdateAccounts []models.Account
			now := time.Now()

			for _, account := range accounts {
				checkInterval := time.Duration(userSettings.CheckInterval) * time.Minute
				lastCheck := time.Unix(account.LastCheck, 0)

				if now.Sub(lastCheck) < checkInterval {
					logger.Log.Infof("Skipping check for account %s (not due yet)", account.Title)
					continue
				}

				if account.IsCheckDisabled {
					logger.Log.Infof("Skipping check for disabled account: %s", account.Title)
					continue
				}

				accountsToUpdate = append(accountsToUpdate, account)

				if now.Sub(time.Unix(account.LastNotification, 0)).Hours() >= userSettings.NotificationInterval {
					dailyUpdateAccounts = append(dailyUpdateAccounts, account)
				}
			}

			for _, account := range accountsToUpdate {
				var captchaAPIKey string
				if userSettings.PreferredCaptchaProvider == "2captcha" {
					captchaAPIKey = userSettings.TwoCaptchaAPIKey
				} else {
					captchaAPIKey = userSettings.EZCaptchaAPIKey
				}
				status, err := CheckAccount(account.SSOCookie, account.UserID, captchaAPIKey)
				if err != nil {
					logger.Log.WithError(err).Errorf("Error checking account %s", account.Title)
					NotifyAdminWithCooldown(s, fmt.Sprintf("Error checking account %s: %v", account.Title, err), 5*time.Minute)
					continue
				}

				previousStatus := account.LastStatus
				account.LastStatus = status
				account.LastCheck = now.Unix()
				if err := database.DB.Save(&account).Error; err != nil {
					logger.Log.WithError(err).Errorf("Failed to update account %s after check", account.Title)
					continue
				}

				if previousStatus != status {
					HandleStatusChange(s, account, status, userSettings)
				}

				// Check for cookie expiration
				if !account.IsExpiredCookie {
					timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
					if err == nil && timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cookieExpirationWarning)*time.Hour {
						if err := NotifyCookieExpiringSoon(s, []models.Account{account}); err != nil {
							logger.Log.WithError(err).Errorf("Failed to send cookie expiration notification for account %s", account.Title)
						}
					}
				}
			}

			if len(dailyUpdateAccounts) > 0 {
				SendConsolidatedDailyUpdate(s, userID, userSettings, dailyUpdateAccounts)
			}

		}(userID, userAccounts)
	}
}

func CheckAndSendNotifications(s *discordgo.Session, userID string) {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return
	}

	now := time.Now()

	// Check for daily update
	if now.Sub(userSettings.LastDailyUpdateNotification) >= time.Duration(userSettings.NotificationInterval)*time.Hour {
		SendConsolidatedDailyUpdate(s, userID, userSettings, nil)
	}

	// Check for cookie expiration
	var accounts []models.Account
	if err := database.DB.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to fetch accounts for user %s", userID)
		return
	}

	for _, account := range accounts {
		if !account.IsExpiredCookie {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err != nil {
				logger.Log.WithError(err).Errorf("Error checking SSO cookie expiration for account %s", account.Title)
				continue
			}
			if timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cookieExpirationWarning)*time.Hour {
				if err := NotifyCookieExpiringSoon(s, []models.Account{account}); err != nil {
					logger.Log.WithError(err).Errorf("Failed to send cookie expiration notification for account %s", account.Title)
				}
			}
		}
	}
}

func CheckAndNotifyCookieExpiration(s *discordgo.Session, account models.Account) error {
	timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
	if err != nil {
		return fmt.Errorf("failed to check SSO cookie expiration: %w", err)
	}

	if timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cookieExpirationWarning)*time.Hour {
		return NotifyCookieExpiringSoon(s, []models.Account{account})
	}

	return nil
}

func sendConsolidatedCookieExpirationWarning(s *discordgo.Session, userID string, expiringAccounts []models.Account, userSettings models.UserSettings) {
	var embedFields []*discordgo.MessageEmbedField

	for _, account := range expiringAccounts {
		timeUntilExpiration, _ := CheckSSOCookieExpiration(account.SSOCookieExpiration)
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   account.Title,
			Value:  fmt.Sprintf("Cookie expires in %s", FormatDuration(timeUntilExpiration)),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "SSO Cookie Expiration Warning",
		Description: "The following accounts have SSO cookies that will expire soon:",
		Color:       0xFFA500, // Orange color for warning
		Fields:      embedFields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	err := SendNotification(s, expiringAccounts[0], embed, "", "cookie_expiring_soon")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send consolidated cookie expiration warning for user %s", userID)
	} else {
		userSettings.LastCookieExpirationWarning = time.Now()
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastCookieExpirationWarning for user %s", userID)
		}
	}
}

func checkCookieExpirations(s *discordgo.Session, userID string, userSettings models.UserSettings) {
	var accounts []models.Account
	if err := database.DB.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to fetch accounts for user %s", userID)
		return
	}

	var expiringAccounts []models.Account

	for _, account := range accounts {
		if !account.IsExpiredCookie {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err == nil && timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cookieExpirationWarning)*time.Hour {
				expiringAccounts = append(expiringAccounts, account)
			}
		}
	}

	if len(expiringAccounts) > 0 {
		sendConsolidatedCookieExpirationWarning(s, userID, expiringAccounts, userSettings)
	}
}

func processUserAccounts(s *discordgo.Session, userID string, accounts []models.Account) {
	captchaAPIKey, balance, err := GetUserCaptchaKey(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user captcha key for user %s", userID)
		return
	}

	logger.Log.Infof("User %s captcha balance: %.2f", userID, balance)

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to get user settings for user %s", userID)
		return
	}

	var accountsToUpdate []models.Account
	var dailyUpdateAccounts []models.Account
	var cookieExpiringAccounts []models.Account
	now := time.Now()

	for _, account := range accounts {
		if account.IsCheckDisabled || account.IsPermabanned {
			logger.Log.Infof("Skipping check for disabled account: %s", account.Title)
			continue
		}

		checkInterval := userSettings.CheckInterval
		lastCheck := time.Unix(account.LastCheck, 0)
		if now.Sub(lastCheck).Minutes() >= float64(checkInterval) {
			accountsToUpdate = append(accountsToUpdate, account)
		} else {
			logger.Log.Infof("Skipping check for account %s (not due yet)", account.Title)
		}

		if now.Sub(time.Unix(account.LastNotification, 0)).Hours() >= userSettings.NotificationInterval {
			dailyUpdateAccounts = append(dailyUpdateAccounts, account)
		}

		if !account.IsExpiredCookie {
			timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
			if err == nil && timeUntilExpiration > 0 && timeUntilExpiration <= time.Duration(cookieExpirationWarning)*time.Hour {
				cookieExpiringAccounts = append(cookieExpiringAccounts, account)
			}
		}
	}

	for _, account := range accountsToUpdate {
		go CheckSingleAccount(s, account, captchaAPIKey)
	}

	if len(dailyUpdateAccounts) > 0 {
		go SendConsolidatedDailyUpdate(s, userID, userSettings, dailyUpdateAccounts)
	}

	if len(cookieExpiringAccounts) > 0 {
		go func() {
			if err := NotifyCookieExpiringSoon(s, cookieExpiringAccounts); err != nil {
				logger.Log.WithError(err).Error("Failed to send cookie expiration notifications")
			}
		}()
	}

	CheckAndNotifyBalance(s, userID, balance)
}

func handlePermabannedAccount(s *discordgo.Session, account models.Account) {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - Permanent Ban Status", account.Title),
		Description: fmt.Sprintf("The account %s is still permanently banned. Please remove this account from monitoring using the /removeaccount command.", account.Title),
		Color:       GetColorForStatus(models.StatusPermaban, false, account.IsCheckDisabled),
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	err := SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), "permaban")
	if err != nil {
		if strings.Contains(err.Error(), "bot might have been removed") {
			logger.Log.Warnf("Bot removed for account %s. Considering account inactive.", account.Title)
			account.IsCheckDisabled = true
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to update account status for %s", account.Title)
			}
		} else {
			logger.Log.WithError(err).Errorf("Failed to send permaban update for account %s", account.Title)
		}
	} else {
		account.LastNotification = time.Now().Unix()
		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update LastNotification for account %s", account.Title)
		}
	}
}

func CheckSingleAccount(s *discordgo.Session, account models.Account, captchaAPIKey string) {
	logger.Log.Infof("Checking account: %s", account.Title)

	if account.IsCheckDisabled {
		logger.Log.Infof("Account %s is disabled. Reason: %s", account.Title, account.DisabledReason)
		return
	}

	timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to check SSO cookie expiration for account %s", account.Title)
		handleCheckAccountError(s, account, err)
		return
	} else if timeUntilExpiration > 0 && timeUntilExpiration <= 24*time.Hour {
		if err := NotifyCookieExpiringSoon(s, []models.Account{account}); err != nil {
			logger.Log.WithError(err).Errorf("Failed to send cookie expiration notification for account %s", account.Title)
		}
	}

	result, err := CheckAccount(account.SSOCookie, account.UserID, captchaAPIKey)
	if err != nil {
		logger.Log.WithError(err).Errorf("Error checking account %s", account.Title)
		handleCheckAccountError(s, account, err)
		NotifyAdminWithCooldown(s, fmt.Sprintf("Error checking account %s: %v", account.Title, err), 5*time.Minute)
		return
	}

	account.ConsecutiveErrors = 0
	account.LastSuccessfulCheck = time.Now()

	updateAccountStatus(s, account, result)
}

func handleCheckAccountError(s *discordgo.Session, account models.Account, err error) {
	account.ConsecutiveErrors++
	account.LastErrorTime = time.Now()

	switch {
	case strings.Contains(err.Error(), "Missing Access") || strings.Contains(err.Error(), "Unknown Channel"):
		disableAccount(s, account, "Bot removed from server/channel")
	case strings.Contains(err.Error(), "insufficient balance"):
		disableAccount(s, account, "Insufficient captcha balance")
	case strings.Contains(err.Error(), "invalid captcha API key"):
		disableAccount(s, account, "Invalid captcha API key")
	default:
		if account.ConsecutiveErrors >= maxConsecutiveErrors {
			disableAccount(s, account, fmt.Sprintf("Too many consecutive errors: %v", err))
		} else {
			logger.Log.WithError(err).Errorf("Failed to check account %s: possible expired SSO Cookie", account.Title)
			notifyUserOfCheckError(s, account, err)
		}
	}

	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to update account %s after error", account.Title)
	}
}

func disableAccount(s *discordgo.Session, account models.Account, reason string) {
	account.IsCheckDisabled = true
	account.DisabledReason = reason

	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to disable account %s", account.Title)
		return
	}

	logger.Log.Infof("Account %s has been disabled. Reason: %s", account.Title, reason)

	NotifyUserAboutDisabledAccount(s, account, reason)
}

func handleInvalidCookie(s *discordgo.Session, account models.Account) {
	userSettings, _ := GetUserSettings(account.UserID)
	lastNotification := time.Unix(account.LastCookieNotification, 0)
	now := time.Now()

	timeUntilExpiration, err := CheckSSOCookieExpiration(account.SSOCookieExpiration)
	if err != nil {
		logger.Log.WithError(err).Errorf("Error checking SSO cookie expiration for account %s", account.Title)
		return
	}

	if timeUntilExpiration > 0 && timeUntilExpiration <= 24*time.Hour &&
		(now.Sub(lastNotification).Hours() >= userSettings.CooldownDuration || account.LastCookieNotification == 0) {

		logger.Log.Infof("Account %s SSO cookie is expiring soon", account.Title)
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - SSO Cookie Expiring Soon", account.Title),
			Description: fmt.Sprintf("The SSO cookie for account %s will expire in %s. Please update the cookie using the /updateaccount command before it expires.", account.Title, FormatDuration(timeUntilExpiration)),
			Color:       0xFFA500, // Orange color for warning
			Timestamp:   now.Format(time.RFC3339),
		}

		err := SendNotification(s, account, embed, "", "cookie_expiring_soon")
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to send cookie expiration notification for account %s", account.Title)
		} else {
			DBMutex.Lock()
			account.LastCookieNotification = now.Unix()
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Errorf("Failed to save account changes for account %s", account.Title)
			}
			DBMutex.Unlock()
		}
	} else if timeUntilExpiration <= 0 {
		// Cookie has already expired
		logger.Log.Infof("Account %s has an expired SSO cookie", account.Title)
		account.IsExpiredCookie = true
		if err := database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Errorf("Failed to update expired cookie status for account %s", account.Title)
		}
	} else {
		logger.Log.Infof("Skipping cookie expiration notification for account %s (not expiring soon or cooldown)", account.Title)
	}
}

func updateAccountStatus(s *discordgo.Session, account models.Account, result models.Status) {
	DBMutex.Lock()
	defer DBMutex.Unlock()

	lastStatus := account.LastStatus
	now := time.Now()
	account.LastCheck = now.Unix()
	account.IsExpiredCookie = false
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Errorf("Failed to save account changes for account %s", account.Title)
		return
	}

	if result != lastStatus {
		lastStatusChange := time.Unix(account.LastStatusChange, 0)
		if now.Sub(lastStatusChange).Hours() >= statusChangeCooldown {
			userSettings, _ := GetUserSettings(account.UserID)
			HandleStatusChange(s, account, result, userSettings)
		} else {
			logger.Log.Infof("Skipping status change notification for account %s (cooldown)", account.Title)
		}
	}
}

func ScheduleTempBanNotification(s *discordgo.Session, account models.Account, duration string) {
	parts := strings.Split(duration, ",")
	if len(parts) != 2 {
		logger.Log.Errorf("Invalid duration format for account %s: %s", account.Title, duration)
		return
	}
	days, _ := strconv.Atoi(strings.TrimSpace(strings.Split(parts[0], " ")[0]))
	hours, _ := strconv.Atoi(strings.TrimSpace(strings.Split(parts[1], " ")[0]))

	sleepDuration := time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour

	for remainingTime := sleepDuration; remainingTime > 0; remainingTime -= 24 * time.Hour {
		if remainingTime > 24*time.Hour {
			time.Sleep(24 * time.Hour)
		} else {
			time.Sleep(remainingTime)
		}

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

	result, err := CheckAccount(account.SSOCookie, account.UserID, "")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to check account %s after temporary ban duration", account.Title)
		return
	}

	var embed *discordgo.MessageEmbed
	if result == models.StatusGood {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Lifted", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s has been lifted. The account is now in good standing.", account.Title),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	} else if result == models.StatusPermaban {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Escalated", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s has been escalated to a permanent ban.", account.Title),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s - Temporary Ban Update", account.Title),
			Description: fmt.Sprintf("The temporary ban for account %s is still in effect. Current status: %s", account.Title, result),
			Color:       GetColorForStatus(result, false, account.IsCheckDisabled),
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	}

	err = SendNotification(s, account, embed, fmt.Sprintf("<@%s>", account.UserID), "temp_ban_update")
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send temporary ban update message for account %s", account.Title)
	}
}

func GetColorForStatus(status models.Status, isExpiredCookie bool, isCheckDisabled bool) int {
	if isCheckDisabled {
		return 0xA9A9A9 // Dark Gray for disabled checks
	}
	if isExpiredCookie {
		return 0xFF6347 // Tomato for expired cookie
	}
	switch status {
	case models.StatusPermaban:
		return 0x8B0000 // Dark Red for permanent ban
	case models.StatusShadowban:
		return 0xFFD700 // Gold for shadowban
	case models.StatusTempban:
		return 0xFF8C00 // Dark Orange for temporary ban
	case models.StatusGood:
		return 0x32CD32 // Lime Green for good status
	default:
		return 0x708090 // Slate Gray for unknown status
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

		announcementEmbed := createAnnouncementEmbed()

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

func createAnnouncementEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Important Update: Changes to COD Status Bot",
		Description: "Due to high demand, we've reached our limit of free EZCaptcha tokens. To ensure continued functionality, we're introducing some changes:",
		Color:       0xFFD700, // Gold color
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "What's Changing",
				Value: "• The check ban feature now requires users to provide their own EZCaptcha API key.\n" +
					"• Without an API key, the bot's check ban functionality will be limited.",
			},
			{
				Name: "How to Get Your Own API Key",
				Value: "1. Sign up at [EZ-Captcha](https://dashboard.ez-captcha.com/#/register?inviteCode=uyNrRgWlEKy) using our referral link.\n" +
					"2. Request a free trial of 10,000 tokens.\n" +
					"3. Use the `/setcaptchaservice` command to set your API key in the bot.",
			},
			{
				Name: "Benefits of Using Your Own API Key",
				Value: "• Uninterrupted access to the check ban feature\n" +
					"• Ability to customize check intervals\n" +
					"• Support the bot's development through our referral program",
			},
			{
				Name: "Next Steps",
				Value: "1. Obtain your API key as soon as possible.\n" +
					"2. Set up your key using the `/setcaptchaservice` command.\n" +
					"3. Adjust your check interval preferences if desired.",
			},
			{
				Name:  "Our Commitment",
				Value: "We're actively exploring ways to maintain a free tier for all users. Your support through the referral program directly contributes to this goal.",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Thank you for your understanding and continued support!",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func getChannelForAnnouncement(s *discordgo.Session, userID string, userSettings models.UserSettings) (string, error) {
	if userSettings.NotificationType == "dm" {
		channel, err := s.UserChannelCreate(userID)
		if err != nil {
			logger.Log.WithError(err).Error("Error creating DM channel for global announcement")
			return "", err
		}
		return channel.ID, nil
	}

	var account models.Account
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at DESC").First(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Error finding recent channel for user")
		return "", err
	}
	return account.ChannelID, nil
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

func calculateBanDuration(banEndTime time.Time) string {
	duration := time.Until(banEndTime)
	if duration < 0 {
		duration = 0
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	return fmt.Sprintf("%d days, %d hours", days, hours)
}

func notifyUserOfCheckError(s *discordgo.Session, account models.Account, err error) {
	canSend, checkErr := CheckNotificationCooldown(account.UserID, "error", time.Hour)
	if checkErr != nil {
		logger.Log.WithError(checkErr).Errorf("Failed to check error notification cooldown for user %s", account.UserID)
		return
	}
	if !canSend {
		logger.Log.Infof("Skipping error notification for user %s due to cooldown", account.UserID)
		return
	}

	NotifyAdminWithCooldown(s, fmt.Sprintf("Error checking account %s (ID: %d): %v", account.Title, account.ID, err), 5*time.Minute)

	if isCriticalError(err) {
		channel, err := s.UserChannelCreate(account.UserID)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to create DM channel for user %s", account.UserID)
			return
		}

		embed := &discordgo.MessageEmbed{
			Title: "Critical Account Check Error",
			Description: fmt.Sprintf("There was a critical error checking your account '%s'. "+
				"The bot developer has been notified and will investigate the issue.", account.Title),
			Color:     0xFF0000, // Red color for critical error
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
		if err != nil {
			logger.Log.WithError(err).Errorf("Failed to send critical error notification to user %s", account.UserID)
			return
		}
		sendCriticalErrorNotification(s, account, err)
	}
}

func sendCriticalErrorNotification(s *discordgo.Session, account models.Account, err error) {
	channel, err := s.UserChannelCreate(account.UserID)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to create DM channel for user %s", account.UserID)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "Critical Account Check Error",
		Description: fmt.Sprintf("There was a critical error checking your account '%s'. "+
			"The bot developer has been notified and will investigate the issue.", account.Title),
		Color:     0xFF0000, // Red color for critical error
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err = s.ChannelMessageSendEmbed(channel.ID, embed)
	if err != nil {
		logger.Log.WithError(err).Errorf("Failed to send critical error notification to user %s", account.UserID)
		return
	}
	if updateErr := UpdateNotificationTimestamp(account.UserID, "error"); updateErr != nil {
		logger.Log.WithError(updateErr).Errorf("Failed to update error notification timestamp for user %s", account.UserID)
	}
}
func isCriticalError(err error) bool {
	criticalErrors := []string{
		"invalid captcha API key",
		"insufficient balance",
		"bot removed from server/channel",
	}

	for _, criticalErr := range criticalErrors {
		if strings.Contains(err.Error(), criticalErr) {
			return true
		}
	}
	return false
}
