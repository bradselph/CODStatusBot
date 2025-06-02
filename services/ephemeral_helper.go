package services

import (
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func GetUserEphemeralPreference(userID string) bool {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		return true
	}
	return userSettings.PreferEphemeralResponses
}

func GetInteractionFlags(i *discordgo.InteractionCreate, forceEphemeral bool) discordgo.MessageFlags {
	if forceEphemeral {
		return discordgo.MessageFlagsEphemeral
	}

	userID, err := GetUserID(i)
	if err != nil {
		return discordgo.MessageFlagsEphemeral
	}

	if GetUserEphemeralPreference(userID) {
		return discordgo.MessageFlagsEphemeral
	}

	return 0
}

func RespondWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, forceEphemeral bool) error {
	flags := GetInteractionFlags(i, forceEphemeral)

	responseData := &discordgo.InteractionResponseData{
		Flags: flags,
	}

	if content != "" {
		responseData.Content = content
	}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
	})
}

func RespondWithPreferenceAndComponents(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent, forceEphemeral bool) error {
	flags := GetInteractionFlags(i, forceEphemeral)

	responseData := &discordgo.InteractionResponseData{
		Flags: flags,
	}

	if content != "" {
		responseData.Content = content
	}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	if len(components) > 0 {
		responseData.Components = components
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: responseData,
	})
}

func DeferWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, forceEphemeral bool) error {
	flags := GetInteractionFlags(i, forceEphemeral)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: flags,
		},
	})
}

func FollowupWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent, forceEphemeral bool) (*discordgo.Message, error) {
	flags := GetInteractionFlags(i, forceEphemeral)

	params := &discordgo.WebhookParams{
		Flags: flags,
	}

	if content != "" {
		params.Content = content
	}

	if len(embeds) > 0 {
		params.Embeds = embeds
	}

	if len(components) > 0 {
		params.Components = components
	}

	return s.FollowupMessageCreate(i.Interaction, true, params)
}

func UpdateMessageWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	responseData := &discordgo.InteractionResponseData{}

	if content != "" {
		responseData.Content = content
	}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	if components != nil {
		responseData.Components = components
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: responseData,
	})
}
