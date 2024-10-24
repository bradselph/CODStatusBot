package setcaptchaservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
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
							Label:       "API Key",
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

		var settings models.UserSettings
		if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&settings).Error; err != nil {
			logger.Log.WithError(err).Error("Error getting/creating user settings")
			respondToInteraction(s, i, "Error updating settings. Please try again.")
			return
		}

		settings.PreferredCaptchaProvider = provider
		settings.CheckInterval = 15
		settings.NotificationInterval = 12
		settings.CustomSettings = true

		if provider == "ezcaptcha" {
			settings.EZCaptchaAPIKey = apiKey
			settings.TwoCaptchaAPIKey = ""
		} else {
			settings.TwoCaptchaAPIKey = apiKey
			settings.EZCaptchaAPIKey = ""
		}

		if err := database.DB.Save(&settings).Error; err != nil {
			logger.Log.WithError(err).Error("Error saving user settings")
			respondToInteraction(s, i, "Error saving settings. Please try again.")
			return
		}

		logger.Log.Infof("Valid %s key set for user: %s. Balance: %.2f points", provider, userID, balance)
		message = fmt.Sprintf("Your %s API key has been set successfully. Your current balance is %.2f points. You now have access to faster check intervals and no rate limits!", provider, balance)
	} else {
		if err := services.RemoveCaptchaKey(userID); err != nil {
			logger.Log.WithError(err).Error("Error removing API key")
			respondToInteraction(s, i, "Error removing API key. Please try again.")
			return
		}
		message = "Your API key has been removed. The bot's default API key will be used. Your check interval and notification settings have been reset to default values."
	}

	respondToInteraction(s, i, message)

	go func() {
		time.Sleep(5 * time.Second)
		services.CheckAndNotifyBalance(s, userID, 0)
	}()
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
