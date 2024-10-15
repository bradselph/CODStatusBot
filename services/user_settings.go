package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	api2captcha "CODStatusBot/api2captcha"
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
)

var defaultSettings models.UserSettings

func init() {
	checkInterval, err := strconv.Atoi(os.Getenv("CHECK_INTERVAL"))
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse CHECK_INTERVAL, using default of 15 minutes")
		checkInterval = 15
	}

	defaultInterval, err := strconv.ParseFloat(os.Getenv("NOTIFICATION_INTERVAL"), 64)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse NOTIFICATION_INTERVAL from .env, using default of 24 hours")
		defaultInterval = 24

		defaultSettings.NotificationInterval = defaultInterval

		defaultInterval = 24
	}

	cooldownDuration, err := strconv.ParseFloat(os.Getenv("COOLDOWN_DURATION"), 64)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse COOLDOWN_DURATION, using default of 6 hours")
		cooldownDuration = 6
	}
	defaultSettings.NotificationInterval = defaultInterval

	statusChangeCooldown, err := strconv.ParseFloat(os.Getenv("STATUS_CHANGE_COOLDOWN"), 64)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse STATUS_CHANGE_COOLDOWN, using default of 1 hour")
		statusChangeCooldown = 1
	}

	defaultSettings = models.UserSettings{
		CheckInterval:        checkInterval,
		NotificationInterval: defaultInterval,
		CooldownDuration:     cooldownDuration,
		StatusChangeCooldown: statusChangeCooldown,
		NotificationType:     "channel",
	}

	logger.Log.Infof("Default settings loaded: CheckInterval=%d, NotificationInterval=%.2f, CooldownDuration=%.2f, StatusChangeCooldown=%.2f",
		defaultSettings.CheckInterval, defaultSettings.NotificationInterval, defaultSettings.CooldownDuration, defaultSettings.StatusChangeCooldown)

}

func GetUserSettings(userID string) (models.UserSettings, error) {
	logger.Log.Infof("Getting user settings for user: %s", userID)
	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&settings)
	if result.Error != nil {
		return models.UserSettings{}, fmt.Errorf("error getting user settings: %w", result.Error)
	}

	// If the user doesn't have custom settings, use default settings.
	if settings.CheckInterval == 0 {
		settings.CheckInterval = defaultSettings.CheckInterval
	}
	if settings.NotificationInterval == 0 {
		settings.NotificationInterval = defaultSettings.NotificationInterval
	}
	if settings.CooldownDuration == 0 {
		settings.CooldownDuration = defaultSettings.CooldownDuration
	}
	if settings.StatusChangeCooldown == 0 {
		settings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown
	}
	if settings.NotificationType == "" {
		settings.NotificationType = defaultSettings.NotificationType
	}

	logger.Log.Infof("Got user settings for user: %s", userID)
	return settings, nil
}

func GetUserCaptchaKey(userID string) (string, float64, error) {
	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).First(&settings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings")
		return "", 0, result.Error
	}

	var apiKey string
	var provider string

	if settings.PreferredCaptchaProvider == "2captcha" && settings.TwoCaptchaAPIKey != "" {
		apiKey = settings.TwoCaptchaAPIKey
		provider = "2captcha"
	} else if settings.EZCaptchaAPIKey != "" {
		apiKey = settings.EZCaptchaAPIKey
		provider = "ezcaptcha"
	} else {
		// Use default EZCaptcha key if user hasn't set a custom key
		apiKey = os.Getenv("EZCAPTCHA_CLIENT_KEY")
		provider = "ezcaptcha"
	}

	isValid, balance, err := ValidateCaptchaKey(apiKey, provider)
	if err != nil {
		return "", 0, err
	}
	if !isValid {
		return "", 0, fmt.Errorf("invalid %s API key", provider)
	}

	return apiKey, balance, nil
}

func SetUserCaptchaKey(userID string, apiKey string, provider string) error {
	if !isValidUserID(userID) {
		logger.Log.Error("Invalid userID provided")
		return fmt.Errorf("invalid userID")
	}

	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&settings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error retrieving user settings")
		return result.Error
	}

	if apiKey != "" {
		switch provider {
		case "ezcaptcha":
			settings.EZCaptchaAPIKey = apiKey
		case "2captcha":
			settings.TwoCaptchaAPIKey = apiKey
		default:
			logger.Log.Errorf("Invalid captcha provider %s for user %s", provider, userID)
			return fmt.Errorf("invalid captcha provider")
		}
		settings.PreferredCaptchaProvider = provider

		// Enable custom settings for valid API key
		settings.CheckInterval = 15        // Allow more frequent checks, e.g., every 15 minutes
		settings.NotificationInterval = 12 // Allow more frequent notifications, e.g., every 12 hours

		logger.Log.Infof("Setting %s key for user %s", provider, userID)
	} else {
		// Reset to default settings when API key is removed
		settings.EZCaptchaAPIKey = ""
		settings.TwoCaptchaAPIKey = ""
		settings.PreferredCaptchaProvider = "ezcaptcha" // Reset to default provider
		settings.CheckInterval = defaultSettings.CheckInterval
		settings.NotificationInterval = defaultSettings.NotificationInterval
		settings.CooldownDuration = defaultSettings.CooldownDuration
		settings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown

		logger.Log.Infof("Removing captcha key for user %s. Resetting to default settings", userID)
	}

	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Errorf("Error saving settings for user %s", userID)
		return fmt.Errorf("error saving user settings: %w", err)
	}

	logger.Log.Infof("Successfully updated captcha key and settings for user %s", userID)
	return nil
}

func GetCaptchaSolver(userID string) (CaptchaSolver, error) {
	apiKey, _, err := GetUserCaptchaKey(userID)
	if err != nil {
		return nil, err
	}

	userSettings, err := GetUserSettings(userID)
	if err != nil {
		return nil, err
	}

	provider := userSettings.PreferredCaptchaProvider
	if provider == "" {
		provider = "ezcaptcha" // Default to EZCaptcha if not set
	}

	return NewCaptchaSolver(apiKey, provider)
}

func isValidUserID(userID string) bool {
	// Check if userID consists of only digits (Discord user IDs are numeric).
	if len(userID) < 17 || len(userID) > 20 {
		return false
	}
	for _, char := range userID {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func GetDefaultSettings() (models.UserSettings, error) {
	return defaultSettings, nil
}

func RemoveCaptchaKey(userID string) error {
	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).First(&settings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error removing apikey in settings")
		return result.Error
	}

	settings.EZCaptchaAPIKey = ""
	settings.TwoCaptchaAPIKey = ""
	settings.PreferredCaptchaProvider = "ezcaptcha" // Reset to default provider
	settings.CheckInterval = defaultSettings.CheckInterval
	settings.NotificationInterval = defaultSettings.NotificationInterval
	settings.CooldownDuration = defaultSettings.CooldownDuration
	settings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown

	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		return err
	}

	logger.Log.Infof("Removed captcha key and reset settings for user: %s", userID)
	return nil
}

func UpdateUserSettings(userID string, newSettings models.UserSettings) error {
	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&settings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error updating user settings")
		return result.Error
	}

	// User can only update settings if they have a valid API key.
	if settings.EZCaptchaAPIKey != "" || settings.TwoCaptchaAPIKey != "" {
		if newSettings.CheckInterval != 0 {
			settings.CheckInterval = newSettings.CheckInterval
		}
		if newSettings.NotificationInterval != 0 {
			settings.NotificationInterval = newSettings.NotificationInterval
		}
		if newSettings.CooldownDuration != 0 {
			settings.CooldownDuration = newSettings.CooldownDuration
		}
		if newSettings.StatusChangeCooldown != 0 {
			settings.StatusChangeCooldown = newSettings.StatusChangeCooldown
		}
	}

	// Allow updating notification type regardless of API key
	if newSettings.NotificationType != "" {
		settings.NotificationType = newSettings.NotificationType
	}

	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Error("Error updating user settings")
		return err
	}

	logger.Log.Infof("Updated settings for user: %s", userID)
	return nil
}

func CheckCaptchaKeyValidity(captchaKey string) (bool, float64, error) {
	url := "https://api.ez-captcha.com/getBalance"
	payload := map[string]string{
		"clientKey": captchaKey,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, 0, fmt.Errorf("failed to marshal JSON payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, 0, fmt.Errorf("failed to send getBalance request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.WithError(err).Error("Error closing response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, fmt.Errorf("failed to read response body: %v", err)
	}

	var result struct {
		ErrorId int     `json:"errorId"`
		Balance float64 `json:"balance"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if result.ErrorId != 0 {
		return false, 0, nil
	}

	return true, result.Balance, nil
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
			if user.EZCaptchaAPIKey != "" {
				_, balance, err := ValidateCaptchaKey(user.EZCaptchaAPIKey, "ezcaptcha")
				if err != nil {
					logger.Log.WithError(err).Errorf("Failed to validate EZCaptcha key for user %s", user.UserID)
					continue
				}
				CheckAndNotifyBalance(s, user.UserID, balance)
			} else if user.TwoCaptchaAPIKey != "" {
				_, balance, err := ValidateCaptchaKey(user.TwoCaptchaAPIKey, "2captcha")
				if err != nil {
					logger.Log.WithError(err).Errorf("Failed to validate 2captcha key for user %s", user.UserID)
					continue
				}
				CheckAndNotifyBalance(s, user.UserID, balance)
			}
		}
	}
}

func ValidateEZCaptchaKey(apiKey string) (bool, float64, error) {
	return CheckCaptchaKeyValidity(apiKey)
}

func ValidateTwoCaptchaKey(apiKey string) (bool, float64, error) {
	client := api2captcha.NewClient(apiKey)
	balance, err := client.GetBalance()
	if err != nil {
		return false, 0, err
	}
	return true, balance, nil
}
