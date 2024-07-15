package removeaccountnew

import (
	"CODStatusBot/database"
	"CODStatusBot/logger"
	"CODStatusBot/models"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

func RegisterCommand(s *discordgo.Session, guildID string) {
	command := &discordgo.ApplicationCommand{
		Name:        "removeaccountnew",
		Description: "Remove a monitored account",
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, command)
	if err != nil {
		logger.Log.WithError(err).Error("Error creating removeaccountnew command")
	}
}

func UnregisterCommand(s *discordgo.Session, guildID string) {
	commands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands")
		return
	}

	for _, cmd := range commands {
		if cmd.Name == "removeaccountnew" {
			err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
			if err != nil {
				logger.Log.WithError(err).Error("Error deleting removeaccountnew command")
			}
			return
		}
	}
}

func CommandRemoveAccountNew(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	guildID := i.GuildID

	var accounts []models.Account
	result := database.DB.Where("user_id = ? AND guild_id = ?", userID, guildID).Find(&accounts)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching user accounts")
		respondToInteraction(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		respondToInteraction(s, i, "You don't have any monitored accounts to remove.")
		return
	}

	accountList := "Your accounts:\n"
	for _, account := range accounts {
		accountList += fmt.Sprintf("• %s (Status: %s)\n", account.Title, account.LastStatus)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "remove_account_modal",
			Title:    "Remove Account",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "account_title",
							Label:       "Enter the title of the account to remove",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter the account title",
							Required:    true,
							MinLength:   1,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "confirmation",
							Label:       "Type 'CONFIRM' to remove this account",
							Style:       discordgo.TextInputShort,
							Placeholder: "CONFIRM",
							Required:    true,
							MinLength:   7,
							MaxLength:   7,
						},
					},
				},
			},
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Error responding with modal")
		return
	}

	// Send the account list as a follow-up message
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: accountList,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error sending follow-up message with account list")
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	var accountTitle, confirmation string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if textInput, ok := rowComp.(*discordgo.TextInput); ok {
					switch textInput.CustomID {
					case "account_title":
						accountTitle = strings.TrimSpace(textInput.Value)
					case "confirmation":
						confirmation = strings.TrimSpace(textInput.Value)
					}
				}
			}
		}
	}

	if accountTitle == "" {
		respondToInteraction(s, i, "Error: No account title provided.")
		return
	}

	if confirmation != "CONFIRM" {
		respondToInteraction(s, i, "Account removal cancelled. The confirmation was not correct.")
		return
	}

	var account models.Account
	result := database.DB.Where("title = ? AND user_id = ? AND guild_id = ?", accountTitle, i.Member.User.ID, i.GuildID).First(&account)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to remove it.")
		return
	}

	// Start a transaction
	tx := database.DB.Begin()

	// Delete associated bans
	if err := tx.Where("account_id = ?", account.ID).Delete(&models.Ban{}).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Error deleting associated bans")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	// Delete the account
	if err := tx.Delete(&account).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Error deleting account")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).Error("Error committing transaction")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	respondToInteraction(s, i, fmt.Sprintf("Account '%s' has been successfully removed from the database. This action cannot be undone.", account.Title))
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
