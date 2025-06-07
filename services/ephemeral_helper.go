package services

import (
	"github.com/bradselph/CODStatusBot/configuration"
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
	cfg := configuration.Get()

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
		if cfg.ComponentsV2.Enabled {
			responseData.Components = components
			responseData.Flags |= discordgo.MessageFlagsIsComponentsV2
		} else {
			var wrappedComponents []discordgo.MessageComponent
			for i := 0; i < len(components); i += 5 {
				end := i + 5
				if end > len(components) {
					end = len(components)
				}
				row := discordgo.ActionsRow{
					Components: components[i:end],
				}
				wrappedComponents = append(wrappedComponents, row)
			}
			responseData.Components = wrappedComponents
		}
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
	cfg := configuration.Get()

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
		if cfg.ComponentsV2.Enabled {
			params.Components = components
			params.Flags |= discordgo.MessageFlagsIsComponentsV2
		} else {
			var wrappedComponents []discordgo.MessageComponent
			for i := 0; i < len(components); i += 5 {
				end := i + 5
				if end > len(components) {
					end = len(components)
				}
				row := discordgo.ActionsRow{
					Components: components[i:end],
				}
				wrappedComponents = append(wrappedComponents, row)
			}
			params.Components = wrappedComponents
		}
	}

	return s.FollowupMessageCreate(i.Interaction, true, params)
}

func UpdateMessageWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	cfg := configuration.Get()
	responseData := &discordgo.InteractionResponseData{}

	if content != "" {
		responseData.Content = content
	}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	if components != nil {
		if cfg.ComponentsV2.Enabled && len(components) > 0 {
			responseData.Components = components
			responseData.Flags = discordgo.MessageFlagsIsComponentsV2
		} else {
			var wrappedComponents []discordgo.MessageComponent
			for i := 0; i < len(components); i += 5 {
				end := i + 5
				if end > len(components) {
					end = len(components)
				}
				row := discordgo.ActionsRow{
					Components: components[i:end],
				}
				wrappedComponents = append(wrappedComponents, row)
			}
			responseData.Components = wrappedComponents
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: responseData,
	})
}

func ValidateEmbedLimits(embed *discordgo.MessageEmbed) *discordgo.MessageEmbed {
	cfg := configuration.Get()

	if embed == nil {
		return embed
	}

	if len(embed.Description) > cfg.Message.EmbedDescLimit {
		embed.Description = embed.Description[:cfg.Message.EmbedDescLimit-3] + "..."
	}

	if len(embed.Fields) > cfg.Message.MaxEmbedFields {
		embed.Fields = embed.Fields[:cfg.Message.MaxEmbedFields]
	}

	for i, field := range embed.Fields {
		if len(field.Value) > 1024 {
			embed.Fields[i].Value = field.Value[:1021] + "..."
		}
		if len(field.Name) > 256 {
			embed.Fields[i].Name = field.Name[:253] + "..."
		}
	}

	return embed
}

func ValidateMessageLimits(content string) string {
	cfg := configuration.Get()

	if len(content) > cfg.Message.MaxLength {
		return content[:cfg.Message.MaxLength-3] + "..."
	}

	return content
}
