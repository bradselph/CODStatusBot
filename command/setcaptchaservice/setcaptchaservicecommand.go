package setcaptchaservice

import (
	"CODStatusBot/database"
	"CODStatusBot/models"
	"fmt"
	"strings"
	"time"

	"CODStatusBot/logger"
	"CODStatusBot/services"
	"CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

func CommandSetCaptchaService(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "set_captcha_service_modal",
			Title:    "Set Captcha Service API Key",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "captcha_provider",
							Label:       "Captcha Provider (ezcaptcha or 2captcha)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter 'ezcaptcha' or '2captcha'",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "api_key",
							Label:       "Enter your Captcha API key",
							Style:       discordgo.TextInputShort,
							Placeholder: "Leave blank to use bot's default key",
							Required:    false,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with modal")
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	var provider, apiKey string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "captcha_provider":
						provider = strings.ToLower(utils.SanitizeInput(strings.TrimSpace(textInput.Value)))
					case "api_key":
						apiKey = utils.SanitizeInput(strings.TrimSpace(textInput.Value))
					}
				}
			}
		}
	}

	logger.Log.Infof("Received setcaptchaservice command. Provider: %s, API Key length: %d", provider, len(apiKey))

	if provider != "ezcaptcha" && provider != "2captcha" {
		logger.Log.Errorf("Invalid captcha provider: %s", provider)
		respondToInteraction(s, i, "Invalid captcha provider. Please enter 'ezcaptcha' or '2captcha'.")
		return
	}

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var message string
	if apiKey != "" {
		isValid, balance, err := services.ValidateCaptchaKey(apiKey, provider)
		if err != nil {
			logger.Log.WithError(err).Errorf("Error validating %s API key for user %s", provider, userID)
			respondToInteraction(s, i, fmt.Sprintf("Error validating the %s API key: %v. Please try again.", provider, err))
			return
		}
		if !isValid {
			logger.Log.Errorf("Invalid %s API key provided by user %s", provider, userID)
			respondToInteraction(s, i, fmt.Sprintf("The provided %s API key is invalid. Please check and try again.", provider))
			return
		}

		logger.Log.Infof("Valid %s key set for user: %s. Balance: %.2f points", provider, userID, balance)
		message = fmt.Sprintf("Your %s API key has been set successfully. Your current balance is %.2f points. You now have access to faster check intervals and no rate limits!", provider, balance)
	} else {
		message = fmt.Sprintf("Your %s API key has been removed. The bot's default API key will be used. Your check interval and notification settings have been reset to default values.", provider)
	}

	err := services.SetUserCaptchaKey(userID, apiKey, provider)
	if err != nil {
		logger.Log.WithError(err).Errorf("Error setting %s API key for user %s", provider, userID)
		respondToInteraction(s, i, fmt.Sprintf("Error setting %s API key: %v. Please try again.", provider, err))
		return
	}

	respondToInteraction(s, i, message)
}

func validateCaptchaKey(apiKey, provider string) (bool, float64, error) {
	switch provider {
	case "ezcaptcha":
		return services.ValidateEZCaptchaKey(apiKey)
	case "2captcha":
		isValid, balance, err := services.ValidateTwoCaptchaKey(apiKey)
		if err != nil {
			if strings.Contains(err.Error(), "CAPCHA_NOT_READY") || strings.Contains(err.Error(), "ERROR_CAPTCHA_UNSOLVABLE") {
				logger.Log.WithError(err).Warn("Temporary 2captcha error, retrying...")
				time.Sleep(2 * time.Second)
				return services.ValidateTwoCaptchaKey(apiKey)
			}
			logger.Log.WithError(err).Error("Error validating 2captcha key")
			return false, 0, fmt.Errorf("2captcha validation error: %w", err)
		}
		if !isValid {
			logger.Log.Error("2captcha key is invalid")
			return false, 0, nil
		}
		return true, balance, nil
	default:
		return false, 0, fmt.Errorf("unsupported captcha provider")
	}
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

// it may be possible to remove this function if it's not used
func updateUserCaptchaSettings(userID, apiKey, provider string) error {
	var settings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&settings)
	if result.Error != nil {
		return result.Error
	}

	settings.PreferredCaptchaProvider = provider
	if provider == "ezcaptcha" {
		settings.EZCaptchaAPIKey = apiKey
		settings.TwoCaptchaAPIKey = "" // Clear the other provider's key
	} else if provider == "2captcha" {
		settings.TwoCaptchaAPIKey = apiKey
		settings.EZCaptchaAPIKey = "" // Clear the other provider's key
	}

	if apiKey != "" {
		// Enable custom settings for valid API key
		settings.CheckInterval = 15        // Allow more frequent checks, e.g., every 15 minutes
		settings.NotificationInterval = 12 // Allow more frequent notifications, e.g., every 12 hours
	} else {
		// Reset to default settings when API key is removed
		defaultSettings, _ := services.GetDefaultSettings()
		settings.CheckInterval = defaultSettings.CheckInterval
		settings.NotificationInterval = defaultSettings.NotificationInterval
	}

	return database.DB.Save(&settings).Error
}
