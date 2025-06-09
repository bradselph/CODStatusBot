package services

import (
	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
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

	if cfg.ComponentsV2.Enabled && len(components) > 0 {
		v2Components, legacyComponents := separateComponentsByVersion(components)

		if len(v2Components) > 0 {
			v2Response := &discordgo.InteractionResponseData{
				Flags: flags | discordgo.MessageFlagsIsComponentsV2,
			}

			if content != "" {
				v2Response.Content = content
			}

			if len(embeds) > 0 {
				v2Response.Embeds = embeds
			}

			v2Response.Components = v2Components

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: v2Response,
			})

			if err == nil {
				return nil
			}

			logger.Log.WithError(err).Warn("Components v2 failed, falling back to legacy components")
		}

		components = legacyComponents
	}

	responseData := &discordgo.InteractionResponseData{
		Flags: flags,
	}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	if len(components) > 0 {
		var wrappedComponents []discordgo.MessageComponent
		for j := 0; j < len(components); j += 5 {
			end := j + 5
			if end > len(components) {
				end = len(components)
			}
			row := discordgo.ActionsRow{
				Components: components[j:end],
			}
			wrappedComponents = append(wrappedComponents, row)
		}
		responseData.Components = wrappedComponents
	}

	if content != "" {
		responseData.Content = content
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

	if cfg.ComponentsV2.Enabled && len(components) > 0 {
		v2Components, legacyComponents := separateComponentsByVersion(components)

		if len(v2Components) > 0 {
			v2Params := &discordgo.WebhookParams{
				Flags: flags | discordgo.MessageFlagsIsComponentsV2,
			}

			if content != "" {
				v2Params.Content = content
			}

			if len(embeds) > 0 {
				v2Params.Embeds = embeds
			}

			v2Params.Components = v2Components

			msg, err := s.FollowupMessageCreate(i.Interaction, true, v2Params)
			if err == nil {
				return msg, nil
			}

			logger.Log.WithError(err).Warn("Components v2 failed, falling back to legacy components")
		}

		components = legacyComponents
	}

	params := &discordgo.WebhookParams{
		Flags: flags,
	}

	if len(embeds) > 0 {
		params.Embeds = embeds
	}

	if len(components) > 0 {
		var wrappedComponents []discordgo.MessageComponent
		for j := 0; j < len(components); j += 5 {
			end := j + 5
			if end > len(components) {
				end = len(components)
			}
			row := discordgo.ActionsRow{
				Components: components[j:end],
			}
			wrappedComponents = append(wrappedComponents, row)
		}
		params.Components = wrappedComponents
	}

	if content != "" {
		params.Content = content
	}

	return s.FollowupMessageCreate(i.Interaction, true, params)
}

func UpdateMessageWithPreference(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embeds []*discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	cfg := configuration.Get()

	if cfg.ComponentsV2.Enabled && len(components) > 0 {
		v2Components, legacyComponents := separateComponentsByVersion(components)

		if len(v2Components) > 0 {
			v2Response := &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsIsComponentsV2,
			}

			if content != "" {
				v2Response.Content = content
			}

			if len(embeds) > 0 {
				v2Response.Embeds = embeds
			}

			v2Response.Components = v2Components

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: v2Response,
			})

			if err == nil {
				return nil
			}

			logger.Log.WithError(err).Warn("Components v2 failed, falling back to legacy components")
		}

		components = legacyComponents
	}

	responseData := &discordgo.InteractionResponseData{}

	if len(embeds) > 0 {
		responseData.Embeds = embeds
	}

	if components != nil && len(components) > 0 {
		var wrappedComponents []discordgo.MessageComponent
		for j := 0; j < len(components); j += 5 {
			end := j + 5
			if end > len(components) {
				end = len(components)
			}
			row := discordgo.ActionsRow{
				Components: components[j:end],
			}
			wrappedComponents = append(wrappedComponents, row)
		}
		responseData.Components = wrappedComponents
	}

	if content != "" {
		responseData.Content = content
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

func CreateV2Button(label, customID string, style discordgo.ButtonStyle) discordgo.MessageComponent {
	return discordgo.Button{
		Label:    label,
		Style:    style,
		CustomID: customID,
	}
}

func CreateLegacyButton(label, customID string, style discordgo.ButtonStyle) discordgo.MessageComponent {
	return discordgo.Button{
		Label:    label,
		Style:    style,
		CustomID: customID,
	}
}

func CreateComponentsBasedOnConfig(components []discordgo.MessageComponent) []discordgo.MessageComponent {
	cfg := configuration.Get()
	if cfg.ComponentsV2.Enabled {
		return components
	}
	return components
}

func ShouldUseComponentsV2() bool {
	cfg := configuration.Get()
	return cfg.ComponentsV2.Enabled
}

func separateComponentsByVersion(components []discordgo.MessageComponent) ([]discordgo.MessageComponent, []discordgo.MessageComponent) {
	var v2Components []discordgo.MessageComponent
	var legacyComponents []discordgo.MessageComponent

	for _, component := range components {
		compType := component.Type()
		if isV2ComponentType(compType) {
			v2Components = append(v2Components, component)
		} else {
			legacyComponents = append(legacyComponents, component)
		}
	}

	return v2Components, legacyComponents
}

func isV2ComponentType(componentType discordgo.ComponentType) bool {
	allowedV2Types := map[discordgo.ComponentType]bool{
		discordgo.ActionsRowComponent: true,
	}
	switch componentType {
	case 9:
		return true
	case 10:
		return true
	case 12:
		return true
	case 13:
		return true
	case 14:
		return true
	case 17:
		return true
	}

	return allowedV2Types[componentType]
}
