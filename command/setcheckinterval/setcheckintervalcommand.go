package setcheckinterval

import (
	"fmt"
	"strconv"
	"strings"

	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"CODStatusBot/services"
	"CODStatusBot/utils"

	"github.com/bwmarrin/discordgo"
)

func CommandSetCheckInterval(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching your settings. Please try again.")
		return
	}

	if userSettings.EZCaptchaAPIKey == "" && userSettings.TwoCaptchaAPIKey == "" {
		respondToInteraction(s, i, "You need to set your own EZ-Captcha or 2captcha API key using the /setcaptchaservice command before you can modify these settings.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "set_check_interval_modal",
			Title:    "Set User Preferences",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "check_interval",
							Label:       "Check Interval (minutes)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter a number between 1 and 1440 (24 hours)",
							Required:    false,
							MinLength:   0,
							MaxLength:   4,
							Value:       strconv.Itoa(userSettings.CheckInterval),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "notification_interval",
							Label:       "Notification Interval (hours)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter a number between 1 and 24",
							Required:    false,
							MinLength:   0,
							MaxLength:   2,
							Value:       fmt.Sprintf("%.0f", userSettings.NotificationInterval),
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "notification_type",
							Label:       "Notification Type (channel or dm)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter 'channel' or 'dm'",
							Required:    false,
							MinLength:   0,
							MaxLength:   7,
							Value:       userSettings.NotificationType,
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

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching your settings. Please try again.")
		return
	}

	defaultSettings, err := services.GetDefaultSettings()
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching default settings")
		respondToInteraction(s, i, "Error fetching default settings. Please try again.")
		return
	}

	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "check_interval":
						if textInput.Value == "" {
							userSettings.CheckInterval = defaultSettings.CheckInterval
						} else {
							interval, err := strconv.Atoi(utils.SanitizeInput(textInput.Value))
							if err != nil || interval < 1 || interval > 1440 {
								respondToInteraction(s, i, "Invalid check interval. Please enter a number between 1 and 1440.")
								return
							}
							userSettings.CheckInterval = interval
						}
					case "notification_interval":
						if textInput.Value == "" {
							userSettings.NotificationInterval = defaultSettings.NotificationInterval
						} else {
							interval, err := strconv.ParseFloat(utils.SanitizeInput(textInput.Value), 64)
							if err != nil || interval < 1 || interval > 24 {
								respondToInteraction(s, i, "Invalid notification interval. Please enter a number between 1 and 24.")
								return
							}
							userSettings.NotificationInterval = interval
						}
					case "notification_type":
						if textInput.Value == "" {
							userSettings.NotificationType = defaultSettings.NotificationType
						} else {
							notificationType := strings.ToLower(utils.SanitizeInput(textInput.Value))
							if notificationType != "channel" && notificationType != "dm" {
								respondToInteraction(s, i, "Invalid notification type. Please enter 'channel' or 'dm'.")
								return
							}
							userSettings.NotificationType = notificationType
						}
					}
				}
			}
		}
	}

	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		respondToInteraction(s, i, "Error updating your settings. Please try again.")
		return
	}

	// Update all accounts for this user with the new settings
	result := database.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"notification_type": userSettings.NotificationType,
		})

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error updating user accounts")
		respondToInteraction(s, i, "Error updating accounts with new settings. Please try again.")
		return
	}

	logger.Log.Infof("Updated %d accounts for user %s", result.RowsAffected, userID)

	message := fmt.Sprintf("Your preferences have been updated:\n"+
		"Check Interval: %d minutes\n"+
		"Notification Interval: %.1f hours\n"+
		"Notification Type: %s\n\n"+
		"These settings will be used for all your account checks and notifications.",
		userSettings.CheckInterval, userSettings.NotificationInterval, userSettings.NotificationType)

	respondToInteraction(s, i, message)
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
