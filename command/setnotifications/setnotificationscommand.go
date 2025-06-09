package setnotifications

import (
	"fmt"
	"strings"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

func CommandSetNotifications(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	if userID == "" {
		logger.Log.Error("Could not determine user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var userSettings models.UserSettings
	result := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings")
		respondToInteraction(s, i, "Error retrieving your current settings. Please try again.")
		return
	}

	if result.RowsAffected > 0 {
		userSettings.EnsureMapsInitialized()
		if userSettings.NotificationType == "" {
			userSettings.NotificationType = "channel"
		}
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Error("Error saving default user settings")
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("set_notifications_modal_%s", userID),
			Title:    "Set Notification Preferences",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "notification_type",
							Label:       "Notification Type (channel or dm)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter 'channel' or 'dm'",
							Required:    true,
							MinLength:   2,
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

	parts := strings.Split(data.CustomID, "_")
	if len(parts) < 4 {
		logger.Log.WithField("customID", data.CustomID).Error("Invalid modal custom ID format")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}
	userID := parts[len(parts)-1]

	interactionUserID := getUserID(i)
	if interactionUserID == "" {
		logger.Log.Error("Could not determine interaction user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	if interactionUserID != userID {
		logger.Log.WithField("interactionUserID", interactionUserID).WithField("expectedUserID", userID).Error("User ID mismatch")
		respondToInteraction(s, i, "You are not authorized to modify these settings.")
		return
	}

	var notificationType string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					if textInput.CustomID == "notification_type" {
						input := utils.SanitizeInput(textInput.Value)
						notificationType = strings.ToLower(strings.TrimSpace(input))
					}
				}
			}
		}
	}

	if notificationType == "" {
		respondToInteraction(s, i, "Please enter a notification type.")
		return
	}

	switch notificationType {
	case "channel", "ch", "guild", "server":
		notificationType = "channel"
	case "dm", "direct", "private", "dms":
		notificationType = "dm"
	default:
		respondToInteraction(s, i, "Invalid notification type. Please enter 'channel' or 'dm'.")
		return
	}

	var userSettings models.UserSettings
	result := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting/creating user settings")
		respondToInteraction(s, i, "Error updating settings. Please try again.")
		return
	}

	if result.RowsAffected > 0 {
		userSettings.EnsureMapsInitialized()
	}

	if userSettings.NotificationType == notificationType {
		respondToInteraction(s, i, fmt.Sprintf("Your notification type is already set to %s.", notificationType))
		return
	}

	userSettings.NotificationType = notificationType
	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		respondToInteraction(s, i, "Error saving settings. Please try again.")
		return
	}

	accountResult := database.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"notification_type": notificationType,
		})

	if accountResult.Error != nil {
		logger.Log.WithError(accountResult.Error).Error("Error updating user accounts")
		logger.Log.Warn("User notification type updated but account updates failed")
	}

	accountsUpdated := accountResult.RowsAffected
	logger.Log.Infof("Updated notification preferences for user %s to %s (%d accounts updated)", userID, notificationType, accountsUpdated)

	message := fmt.Sprintf("Your notification preferences have been updated to **%s**.\n", notificationType)
	if accountsUpdated > 0 {
		message += fmt.Sprintf("Updated %d account(s) with new notification settings.", accountsUpdated)
	} else {
		message += "No accounts found to update - your new setting will apply to future accounts."
	}

	message = services.ValidateMessageLimits(message)
	respondToInteraction(s, i, message)
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {

		return i.User.ID
	}
	logger.Log.Error("Interaction doesn't have Member or User")
	return ""
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	var err error
	if i.Type == discordgo.InteractionMessageComponent {
		err = services.UpdateMessageWithPreference(s, i, message, nil, []discordgo.MessageComponent{})
	} else {
		err = services.RespondWithPreference(s, i, message, nil, false)
	}
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
