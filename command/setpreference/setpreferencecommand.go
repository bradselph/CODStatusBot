package setpreference

import (
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func CommandSetPreference(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select your preferred notification type:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Channel",
							Style:    discordgo.PrimaryButton,
							CustomID: "set_preference_channel",
						},
						discordgo.Button{
							Label:    "Direct Message",
							Style:    discordgo.PrimaryButton,
							CustomID: "set_preference_dm",
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with preference selection")
	}
}

func HandlePreferenceSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	var preferenceType string

	switch customID {
	case "set_preference_channel":
		preferenceType = "channel"
	case "set_preference_dm":
		preferenceType = "dm"
	default:
		respondToInteraction(s, i, "Invalid preference type. Please try again.")
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

	// Update all existing accounts for this user
	result := database.DB.Model(&models.Account{}).
		Where("user_id = ?", userID).
		Update("notification_type", preferenceType)

	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error updating user accounts")
		respondToInteraction(s, i, "Error setting preference. Please try again.")
		return
	}

	// Log the amount accounts updated.
	logger.Log.Infof("Updated %d accounts for user %s", result.RowsAffected, userID)

	message := "Your notification preference has been updated for all your accounts. "
	if preferenceType == "channel" {
		message += "You will now receive notifications in the channel."
	} else {
		message += "You will now receive notifications via direct message."
	}

	respondToInteraction(s, i, message)
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	var err error
	if i.Type == discordgo.InteractionMessageComponent {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    message,
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: message,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
