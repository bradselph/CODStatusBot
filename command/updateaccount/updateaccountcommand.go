package updateaccount

import (
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"CODStatusBot/services"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strconv"
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

func RegisterCommand(s *discordgo.Session, guildID string) {
	command := &discordgo.ApplicationCommand{
		Name:        "updateaccount",
		Description: "Update a monitored account's information",
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, command)
	if err != nil {
		logger.Log.WithError(err).Error("Error creating updateaccount command")
	}
}

func UnregisterCommand(s *discordgo.Session, guildID string) {
	commands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands")
		return
	}

	for _, cmd := range commands {
		if cmd.Name == "updateaccount" {
			err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
			if err != nil {
				logger.Log.WithError(err).Error("Error deleting updateaccount command")
			}
			return
		}
	}
}

func CommandUpdateAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching user accounts")
		respondToInteraction(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		respondToInteraction(s, i, "You don't have any monitored accounts to update.")
		return
	}

	options := make([]discordgo.SelectMenuOption, len(accounts))
	for index, account := range accounts {
		options[index] = discordgo.SelectMenuOption{
			Label:       account.Title,
			Value:       strconv.FormatUint(uint64(account.ID), 10),
			Description: fmt.Sprintf("Status: %s, Guild: %s", account.LastStatus, account.GuildID),
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Select an account to update:",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "update_account_select",
							Placeholder: "Choose an account",
							Options:     options,
						},
					},
				},
			},
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Error responding with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondToInteraction(s, i, "No account selected. Please try again.")
		return
	}

	accountID, err := strconv.ParseUint(data.Values[0], 10, 64)
	if err != nil {
		logger.Log.WithError(err).Error("Error converting account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("update_account_modal_%d", accountID),
			Title:    "Update Account",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "new_sso_cookie",
							Label:       "Enter the new SSO cookie",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Enter the new SSO cookie",
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
							MaxLength:   100,
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
	if len(parts) != 3 {
		logger.Log.Error("Invalid custom ID format")
		respondToInteraction(s, i, "An error occurred. Please try again.")
		return
	}

	accountID, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		logger.Log.WithError(err).Error("Error converting account ID from modal custom ID")
		respondToInteraction(s, i, "Error processing your update. Please try again.")
		return
	}

	var newSSOCookie string
	var captchaAPIKey string

	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				switch v := rowComp.(type) {
				case *discordgo.TextInput:
					if v.CustomID == "new_sso_cookie" {
						newSSOCookie = strings.TrimSpace(v.Value)
					} else if v.CustomID == "captcha_api_key" {
						captchaAPIKey = strings.TrimSpace(v.Value)
					}
				}
			}
		}
	}

	if newSSOCookie == "" {
		respondToInteraction(s, i, "Error: New SSO cookie must be provided.")
		return
	}

	// Verify the new SSO cookie
	if !services.VerifySSOCookie(newSSOCookie) {
		respondToInteraction(s, i, "Error: The provided SSO cookie is invalid. Please check and try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to update it.")
		return
	}

	// Verify that the user owns this account
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

	if account.UserID != userID {
		logger.Log.Error("User attempted to update an account they don't own")
		respondToInteraction(s, i, "Error: You don't have permission to update this account.")
		return
	}

	// Update the account, preserving the existing NotificationType
	account.SSOCookie = newSSOCookie
	account.IsExpiredCookie = false // Reset the expired cookie flag
	if captchaAPIKey != "" {
		account.CaptchaAPIKey = captchaAPIKey
	}

	services.DBMutex.Lock()
	if err := database.DB.Save(&account).Error; err != nil {
		services.DBMutex.Unlock()
		logger.Log.WithError(err).Error("Error updating account")
		respondToInteraction(s, i, "Error updating account. Please try again.")
		return
	}
	services.DBMutex.Unlock()

	respondToInteraction(s, i, fmt.Sprintf("Account '%s' has been successfully updated with the new SSO cookie and EZ-Captcha settings.", account.Title))
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
