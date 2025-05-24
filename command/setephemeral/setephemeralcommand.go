package setephemeral

import (
	"fmt"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

func CommandSetEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		respondWithError(s, i, "Failed to process your request")
		return
	}

	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings, models.UserSettings{UserID: userID}).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get user settings")
		respondWithError(s, i, "Failed to retrieve your settings")
		return
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Enable Ephemeral Messages",
					Style:    discordgo.SuccessButton,
					CustomID: "set_ephemeral_enable",
					Emoji: &discordgo.ComponentEmoji{
						Name: "👻",
					},
				},
				discordgo.Button{
					Label:    "Disable Ephemeral Messages",
					Style:    discordgo.DangerButton,
					CustomID: "set_ephemeral_disable",
					Emoji: &discordgo.ComponentEmoji{
						Name: "📢",
					},
				},
			},
		},
	}

	currentStatus := "disabled"
	if userSettings.PreferEphemeralResponses {
		currentStatus = "enabled"
	}

	embed := &discordgo.MessageEmbed{
		Title: "Ephemeral Message Settings",
		Description: fmt.Sprintf("Configure whether bot responses should be ephemeral (only visible to you).\n\n"+
			"**Current Setting:** Ephemeral messages are **%s**\n\n"+
			"**What are ephemeral messages?**\n"+
			"• Ephemeral messages are only visible to you\n"+
			"• They disappear after you close Discord\n"+
			"• Other users cannot see these messages\n"+
			"• Useful for keeping your account information private in public channels",
			currentStatus),
		Color: 0x0099FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📱 Recommended for",
				Value:  "• Public servers\n• Shared channels\n• Privacy-conscious users",
				Inline: true,
			},
			{
				Name:   "📢 Not recommended for",
				Value:  "• Private DMs\n• Private channels\n• When you want to share results",
				Inline: true,
			},
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Failed to respond to interaction")
	}
}

func HandleEphemeralSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		respondWithError(s, i, "Failed to process your request")
		return
	}

	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).FirstOrCreate(&userSettings, models.UserSettings{UserID: userID}).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to get user settings")
		respondWithError(s, i, "Failed to retrieve your settings")
		return
	}

	var newSetting bool
	var responseMessage string

	switch customID {
	case "set_ephemeral_enable":
		newSetting = true
		responseMessage = "Ephemeral messages have been **enabled**!\n\n" +
			"From now on, most bot responses will only be visible to you.\n" +
			"You can change this setting anytime using `/setephemeral`."
	case "set_ephemeral_disable":
		newSetting = false
		responseMessage = "Ephemeral messages have been **disabled**!\n\n" +
			"Bot responses will now be visible to everyone in the channel.\n" +
			"You can change this setting anytime using `/setephemeral`."
	default:
		respondWithError(s, i, "Invalid selection")
		return
	}

	userSettings.PreferEphemeralResponses = newSetting
	userSettings.CustomSettings = true

	if err := database.DB.Save(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update user settings")
		respondWithError(s, i, "Failed to update your settings")
		return
	}

	services.LogCommandExecution("setephemeral", userID, i.GuildID, true, 0, "")

	embed := &discordgo.MessageEmbed{
		Title:       "Settings Updated",
		Description: responseMessage,
		Color:       0x00FF00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "This setting applies to all future bot responses",
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Failed to update message")
	}
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Error",
		Description: message,
		Color:       0xFF0000,
	}

	if i.Type == discordgo.InteractionMessageComponent {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: []discordgo.MessageComponent{},
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
