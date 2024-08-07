package addaccount

import (
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
	"strings"
	"unicode"
)

func sanitizeInput(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ' || r == '-' || r == '_' {
			return r
		}
		return -1
	}, input)
}

func CommandAddAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "add_account_modal",
			Title:    "Add New Account",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "account_title",
							Label:       "Account Title",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter a title for this account",
							Required:    true,
							MinLength:   1,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "sso_cookie",
							Label:       "SSO Cookie",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Enter the SSO cookie for this account",
							Required:    true,
							MinLength:   1,
							MaxLength:   4000,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "captcha_api_key",
							Label:       "EZ-Captcha API Key (optional)",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter your own API key (leave blank to use default)",
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

	title := sanitizeInput(strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
	ssoCookie := strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)
	captchaAPIKey := ""

	// Handle captcha API key
	if len(data.Components) > 2 {
		if textInput, ok := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput); ok {
			captchaAPIKey = strings.TrimSpace(textInput.Value)
		}
	}

	// Verify SSO Cookie
	if !services.VerifySSOCookie(ssoCookie) {
		respondToInteraction(s, i, "Invalid SSO cookie. Please try again with a valid cookie.")
		return
	}

	var userID, guildID string
	if i.Member != nil {
		userID = i.Member.User.ID
		guildID = i.GuildID
	} else if i.User != nil {
		userID = i.User.ID
		// In DMs, we don't have a guildID, so we'll leave it empty
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	// Get the user's current notification preference
	var existingAccount models.Account
	result := database.DB.Where("user_id = ?", userID).First(&existingAccount)

	notificationType := "channel" // Default to channel if no existing preference
	if result.Error == nil {
		notificationType = existingAccount.NotificationType
	}

	// Create new account
	account := models.Account{
		UserID:           userID,
		Title:            title,
		SSOCookie:        ssoCookie,
		GuildID:          guildID,
		ChannelID:        i.ChannelID,
		NotificationType: notificationType,
		CaptchaAPIKey:    captchaAPIKey,
	}

	// Save to database
	result = database.DB.Create(&account)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error creating account")
		respondToInteraction(s, i, "Error creating account. Please try again.")
		return
	}

	respondToInteraction(s, i, "Account added successfully!")
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
