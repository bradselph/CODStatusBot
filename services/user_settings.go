package services

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

var defaultSettings models.UserSettings

func initDefaultSettings() {
	cfg := configuration.Get()
	defaultSettings = models.UserSettings{
		CheckInterval:            cfg.Intervals.Check,
		NotificationInterval:     cfg.Intervals.Notification,
		CooldownDuration:         cfg.Intervals.Cooldown,
		StatusChangeCooldown:     cfg.Intervals.StatusChange,
		NotificationType:         "channel",
		PreferredCaptchaProvider: "capsolver",
		CustomSettings:           false,
	}

	if cfg.CaptchaService.Capsolver.Enabled {
		defaultSettings.PreferredCaptchaProvider = "capsolver"
	} else if cfg.CaptchaService.EZCaptcha.Enabled {
		defaultSettings.PreferredCaptchaProvider = "ezcaptcha"
	} else if cfg.CaptchaService.TwoCaptcha.Enabled {
		defaultSettings.PreferredCaptchaProvider = "2captcha"
	}
}

func GetUserSettings(userID string) (models.UserSettings, error) {
	logger.Log.Infof("Getting user settings for user: %s", userID)

	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&settings)
	if result.Error != nil {
		return models.UserSettings{}, fmt.Errorf("error getting user settings: %w", result.Error)
	}

	// Check if user has custom API key
	hasCustomKey := settings.CapSolverAPIKey != "" ||
		settings.EZCaptchaAPIKey != "" ||
		settings.TwoCaptchaAPIKey != ""

	settings.EnsureMapsInitialized()

	if settings.PreferredCaptchaProvider == "" {
		settings.PreferredCaptchaProvider = "capsolver"
	}

	if settings.LastDailyUpdateNotification.IsZero() {
		settings.LastDailyUpdateNotification = time.Now().Add(-24 * time.Hour)
	}

	if !hasCustomKey || !settings.CustomSettings {
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
	}

	if settings.PreferredCaptchaProvider == "" {
		settings.PreferredCaptchaProvider = "capsolver"
	}

	settings.CustomSettings = hasCustomKey

	settings.EnsureMapsInitialized()

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

	cfg := configuration.Get()

	switch settings.PreferredCaptchaProvider {
	case "capsolver":
		if !cfg.CaptchaService.Capsolver.Enabled {
			logger.Log.Warn("Capsolver service disabled, checking for alternative services")

			if settings.EZCaptchaAPIKey != "" {
				settings.PreferredCaptchaProvider = "ezcaptcha"
				if err := database.DB.Save(&settings).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to update preferred provider")
				}
				return settings.EZCaptchaAPIKey, 0, nil
			}
			if settings.TwoCaptchaAPIKey != "" {
				settings.PreferredCaptchaProvider = "2captcha"
				if err := database.DB.Save(&settings).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to update preferred provider")
				}
				return settings.TwoCaptchaAPIKey, 0, nil
			}
			return "", 0, fmt.Errorf("capsolver service is currently disabled")
		}

		if settings.CapSolverAPIKey != "" {
			isValid, balance, err := ValidateCaptchaKey(settings.CapSolverAPIKey, "capsolver")
			if err != nil {
				return "", 0, err
			}
			if !isValid {
				return "", 0, fmt.Errorf("invalid capsolver API key")
			}
			return settings.CapSolverAPIKey, balance, nil
		}

		defaultKey := cfg.CaptchaService.Capsolver.ClientKey
		isValid, balance, err := ValidateCaptchaKey(defaultKey, "capsolver")
		if err != nil {
			return "", 0, err
		}
		if !isValid {
			return "", 0, fmt.Errorf("invalid default capsolver API key")
		}
		return defaultKey, balance, nil

	case "ezcaptcha":
		if !cfg.CaptchaService.EZCaptcha.Enabled {
			if cfg.CaptchaService.Capsolver.Enabled {
				settings.PreferredCaptchaProvider = "capsolver"
				if err := database.DB.Save(&settings).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to update preferred provider")
				}
				return GetUserCaptchaKey(userID)
			}
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

	case "2captcha":
		if !cfg.CaptchaService.TwoCaptcha.Enabled {
			if cfg.CaptchaService.Capsolver.Enabled {
				settings.PreferredCaptchaProvider = "capsolver"
				if err := database.DB.Save(&settings).Error; err != nil {
					logger.Log.WithError(err).Error("Failed to update preferred provider")
				}
				return GetUserCaptchaKey(userID)
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
	}

	// If no custom key is set or no specific provider is selected, use default Capsolver
	if cfg.CaptchaService.Capsolver.Enabled {
		defaultKey := cfg.CaptchaService.Capsolver.ClientKey
		isValid, balance, err := ValidateCaptchaKey(defaultKey, "capsolver")
		if err != nil {
			return "", 0, err
		}
		if !isValid {
			return "", 0, fmt.Errorf("invalid default capsolver API key")
		}
		return defaultKey, balance, nil
	}

	// If Capsolver is disabled, try other enabled services in order of preference
	if cfg.CaptchaService.EZCaptcha.Enabled {
		settings.PreferredCaptchaProvider = "ezcaptcha"
		if err := database.DB.Save(&settings).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to update preferred provider")
		}
		defaultKey := cfg.CaptchaService.EZCaptcha.ClientKey
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
	if defaultSettings.CheckInterval == 0 {
		initDefaultSettings()
	}
	return defaultSettings, nil
}

func RemoveCaptchaKey(userID string) error {
	var settings models.UserSettings
	result := database.DB.Where("user_id = ?", userID).First(&settings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error removing apikey in settings")
		return result.Error
	}

	// Check if user had custom keys before removal
	hadCustomKey := settings.CapSolverAPIKey != "" ||
		settings.EZCaptchaAPIKey != "" ||
		settings.TwoCaptchaAPIKey != ""

	// Get configuration and count accounts only once
	cfg := configuration.Get()
	var accountCount int64
	if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
		return fmt.Errorf("failed to count user accounts: %w", err)
	}

	settings.CapSolverAPIKey = ""
	settings.EZCaptchaAPIKey = ""
	settings.TwoCaptchaAPIKey = ""

	// Reset to default settings
	settings.PreferredCaptchaProvider = defaultSettings.PreferredCaptchaProvider
	settings.CustomSettings = false
	settings.CheckInterval = defaultSettings.CheckInterval
	settings.NotificationInterval = defaultSettings.NotificationInterval
	settings.CooldownDuration = defaultSettings.CooldownDuration
	settings.StatusChangeCooldown = defaultSettings.StatusChangeCooldown

	settings.EnsureMapsInitialized()
	settings.LastCommandTimes["api_key_removed"] = time.Now()

	defaultMax := cfg.RateLimits.DefaultMaxAccounts

	// If user exceeds default limits, send warning
	if int64(defaultMax) < accountCount {
		var accounts []models.Account
		if err := database.DB.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
			logger.Log.WithError(err).Error("Error fetching user accounts while removing API key")
			return err
		}

		// Update all accounts to default notification type
		for _, account := range accounts {
			account.NotificationType = defaultSettings.NotificationType
			if err := database.DB.Save(&account).Error; err != nil {
				logger.Log.WithError(err).Errorf("Error updating account %s settings after API key removal", account.Title)
			}
		}

		// Create warning embed
		embed := &discordgo.MessageEmbed{
			Title: "Account Limit Warning",
			Description: fmt.Sprintf("You currently have %d accounts monitored, which exceeds the default limit of %d accounts.\n"+
				"To continue monitoring all accounts, please add your own Capsolver API key using /setcaptchaservice.",
				accountCount, defaultMax),
			Color: 0xFFA500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Action Required",
					Value:  fmt.Sprintf("Get your Capsolver API key at https://dashboard.capsolver.com/passport/register?inviteCode=6YjROhACQnvP"),
					Inline: false,
				},
				{
					Name:   "Current Accounts",
					Value:  fmt.Sprintf("%d", accountCount),
					Inline: true,
				},
				{
					Name:   "Default Limit",
					Value:  fmt.Sprintf("%d", defaultMax),
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		// Send warning notification
		if len(accounts) > 0 {
			if err := SendNotification(nil, accounts[0], embed, "", "api_key_removal_warning"); err != nil {
				logger.Log.WithError(err).Error("Failed to send API key removal warning")
			}
		}
	}

	// Save updated settings
	if err := database.DB.Save(&settings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		return err
	}

	logger.Log.Infof("Reset settings for user %s (had custom key: %v)", userID, hadCustomKey)
	return nil
}
