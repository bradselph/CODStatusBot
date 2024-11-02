package services

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
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
	}

	cooldownDuration, err := strconv.ParseFloat(os.Getenv("COOLDOWN_DURATION"), 64)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse COOLDOWN_DURATION, using default of 6 hours")
		cooldownDuration = 6
	}

	statusChangeCooldown, err := strconv.ParseFloat(os.Getenv("STATUS_CHANGE_COOLDOWN"), 64)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to parse STATUS_CHANGE_COOLDOWN, using default of 1 hour")
		statusChangeCooldown = 1
	}

	defaultSettings = models.UserSettings{
		CheckInterval:            checkInterval,
		NotificationInterval:     defaultInterval,
		CooldownDuration:         cooldownDuration,
		StatusChangeCooldown:     statusChangeCooldown,
		NotificationType:         "channel",
		PreferredCaptchaProvider: "ezcaptcha",
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
	if settings.PreferredCaptchaProvider == "" {
		settings.PreferredCaptchaProvider = defaultSettings.PreferredCaptchaProvider
	}

	if settings.NotificationTimes == nil {
		settings.NotificationTimes = make(map[string]time.Time)
	}
	if settings.ActionCounts == nil {
		settings.ActionCounts = make(map[string]int)
	}
	if settings.LastActionTimes == nil {
		settings.LastActionTimes = make(map[string]time.Time)
	}
	if settings.LastCommandTimes == nil {
		settings.LastCommandTimes = make(map[string]time.Time)
	}
	if settings.RateLimitExpiration == nil {
		settings.RateLimitExpiration = make(map[string]time.Time)
	}

	if result.RowsAffected > 0 {
		if err := database.DB.Save(&settings).Error; err != nil {
			return settings, fmt.Errorf("error saving default settings: %w", err)
		}
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

	switch settings.PreferredCaptchaProvider {
	case "2captcha":
		if !IsServiceEnabled("2captcha") {
			logger.Log.Warn("Attempt to use disabled 2captcha service")
			if IsServiceEnabled("ezcaptcha") {
				settings.PreferredCaptchaProvider = "ezcaptcha"
				if settings.EZCaptchaAPIKey != "" {
					isValid, balance, err := ValidateCaptchaKey(settings.EZCaptchaAPIKey, "ezcaptcha")
					if err != nil {
						return "", 0, err
					}
					if !isValid {
						return "", 0, fmt.Errorf("invalid ezcaptcha API key")
					}
					database.DB.Save(&settings)
					return settings.EZCaptchaAPIKey, balance, nil
				}
			}
			return "", 0, fmt.Errorf("2captcha service is currently disabled")
		}
		if settings.TwoCaptchaAPIKey != "" {
			isValid, balance, err := ValidateCaptchaKey(settings.TwoCaptchaAPIKey, "2captcha")
			if err != nil {
				return "", 0, err
			}
			if !isValid {
				return "", 0, fmt.Errorf("invalid 2captcha API key")
			}
			return settings.TwoCaptchaAPIKey, balance, nil
		}
	case "ezcaptcha":
		if !IsServiceEnabled("ezcaptcha") {
			return "", 0, fmt.Errorf("ezcaptcha service is currently disabled")
		}
		if settings.EZCaptchaAPIKey != "" {
			isValid, balance, err := ValidateCaptchaKey(settings.EZCaptchaAPIKey, "ezcaptcha")
			if err != nil {
				return "", 0, err
			}
			if !isValid {
				return "", 0, fmt.Errorf("invalid ezcaptcha API key")
			}
			return settings.EZCaptchaAPIKey, balance, nil
		}
	}

	if settings.PreferredCaptchaProvider == "ezcaptcha" && IsServiceEnabled("ezcaptcha") {
		defaultKey := os.Getenv("EZCAPTCHA_CLIENT_KEY")
		isValid, balance, err := ValidateCaptchaKey(defaultKey, "ezcaptcha")
		if err != nil {
			return "", 0, err
		}
		if !isValid {
			return "", 0, fmt.Errorf("invalid default ezcaptcha API key")
		}
		return defaultKey, balance, nil
	}

	return "", 0, fmt.Errorf("no valid API key found for provider %s", settings.PreferredCaptchaProvider)
}

func GetCaptchaSolver(userID string) (CaptchaSolver, error) {
	settings, err := GetUserSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	apiKey, _, err := GetUserCaptchaKey(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user captcha key: %w", err)
	}

	return NewCaptchaSolver(apiKey, settings.PreferredCaptchaProvider)
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
	settings.PreferredCaptchaProvider = "ezcaptcha"
	settings.CheckInterval = defaultSettings.CheckInterval
	settings.NotificationInterval = defaultSettings.NotificationInterval
	settings.CooldownDuration = defaultSettings.CooldownDuration
	settings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown

	if settings.NotificationTimes == nil {
		settings.NotificationTimes = make(map[string]time.Time)
	}
	if settings.ActionCounts == nil {
		settings.ActionCounts = make(map[string]int)
	}
	if settings.LastActionTimes == nil {
		settings.LastActionTimes = make(map[string]time.Time)
	}
	if settings.LastCommandTimes == nil {
		settings.LastCommandTimes = make(map[string]time.Time)
	}
	if settings.RateLimitExpiration == nil {
		settings.RateLimitExpiration = make(map[string]time.Time)
	}

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

	if settings.EZCaptchaAPIKey != "" || settings.TwoCaptchaAPIKey != "" {
		if newSettings.CheckInterval >= 1 && newSettings.CheckInterval <= 1440 {
			settings.CheckInterval = newSettings.CheckInterval
		}
		if newSettings.NotificationInterval >= 1 && newSettings.NotificationInterval <= 24 {
			settings.NotificationInterval = newSettings.NotificationInterval
		}
		if newSettings.CooldownDuration >= 1 && newSettings.CooldownDuration <= 24 {
			settings.CooldownDuration = newSettings.CooldownDuration
		}
		if newSettings.StatusChangeCooldown >= 1 && newSettings.StatusChangeCooldown <= 24 {
			settings.StatusChangeCooldown = newSettings.StatusChangeCooldown
		}
	}

	if newSettings.NotificationType == "channel" || newSettings.NotificationType == "dm" {
		settings.NotificationType = newSettings.NotificationType
	}

	if IsServiceEnabled(newSettings.PreferredCaptchaProvider) {
		settings.PreferredCaptchaProvider = newSettings.PreferredCaptchaProvider
	}

	if newSettings.EZCaptchaAPIKey != "" {
		isValid, _, err := ValidateCaptchaKey(newSettings.EZCaptchaAPIKey, "ezcaptcha")
		if err == nil && isValid {
			settings.EZCaptchaAPIKey = newSettings.EZCaptchaAPIKey
		}
	}

	if newSettings.TwoCaptchaAPIKey != "" {
		isValid, _, err := ValidateCaptchaKey(newSettings.TwoCaptchaAPIKey, "2captcha")
		if err == nil && isValid {
			settings.TwoCaptchaAPIKey = newSettings.TwoCaptchaAPIKey
		}
	}

	if settings.NotificationTimes == nil {
		settings.NotificationTimes = make(map[string]time.Time)
	}
	if settings.ActionCounts == nil {
		settings.ActionCounts = make(map[string]int)
	}
	if settings.LastActionTimes == nil {
		settings.LastActionTimes = make(map[string]time.Time)
	}
	if settings.LastCommandTimes == nil {
		settings.LastCommandTimes = make(map[string]time.Time)
	}
	if settings.RateLimitExpiration == nil {
		settings.RateLimitExpiration = make(map[string]time.Time)
	}

	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		return err
	}

	logger.Log.Infof("Updated settings for user: %s", userID)
	return nil
}
