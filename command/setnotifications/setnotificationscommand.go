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
	if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error getting user settings")
		respondToInteraction(s, i, "Error retrieving your current settings. Please try again.")
		return
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
		logger.Log.Error("Invalid modal custom ID format")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}
	userID := parts[len(parts)-1]

	interactionUserID := getUserID(i)
	if interactionUserID == "" || interactionUserID != userID {
		logger.Log.Error("User ID mismatch or not found")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	var notificationType string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					if textInput.CustomID == "notification_type" {
						notificationType = strings.ToLower(utils.SanitizeInput(textInput.Value))
					}
				}
			}
		}
	}

	if notificationType != "channel" && notificationType != "dm" {
		respondToInteraction(s, i, "Invalid notification type. Please enter 'channel' or 'dm'.")
		return
	}

	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error getting/creating user settings")
		respondToInteraction(s, i, "Error updating settings. Please try again.")
		return
	}

	userSettings.NotificationType = notificationType
	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving user settings")
		respondToInteraction(s, i, "Error saving settings. Please try again.")
		return
	}

	result := database.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"notification_type": notificationType,
		})

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error updating user accounts")
		respondToInteraction(s, i, "Error updating accounts with new settings. Please try again.")
		return
	}

	logger.Log.Infof("Updated notification preferences for user %s to %s", userID, notificationType)
	message := fmt.Sprintf("Your notification preferences have been updated. You will now receive notifications via %s.", notificationType)
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
