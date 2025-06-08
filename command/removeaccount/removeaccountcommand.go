package removeaccount

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"

	"github.com/bwmarrin/discordgo"
)

func CommandRemoveAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		respondToInteraction(s, i, "You don't have any monitored accounts to remove.")
		return
	}

	var components []discordgo.MessageComponent

	for _, account := range accounts {
		components = append(components, discordgo.Button{
			Label:    account.Title,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("remove_account_%d", account.ID),
		})
	}

	err := services.RespondWithPreferenceAndComponents(s, i, "Select an account to remove:", nil, components, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "remove_account_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to remove it.")
		return
	}

	confirmComponents := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Delete",
			Style:    discordgo.DangerButton,
			CustomID: fmt.Sprintf("confirm_remove_%d", account.ID),
		},
		discordgo.Button{
			Label:    "Cancel",
			Style:    discordgo.SecondaryButton,
			CustomID: "cancel_remove",
		},
	}

	err = services.UpdateMessageWithPreference(s, i, fmt.Sprintf("Are you sure you want to remove the account '%s'? This action is permanent and cannot be undone.", account.Title), nil, confirmComponents)
	if err != nil {
		logger.Log.WithError(err).Error("Error showing confirmation buttons")
		err = services.RespondWithPreferenceAndComponents(s, i, fmt.Sprintf("Are you sure you want to remove the account '%s'? This action is permanent and cannot be undone.", account.Title), nil, confirmComponents, false)
		if err != nil {
			logger.Log.WithError(err).Error("Error sending confirmation message")
			respondToInteraction(s, i, "An error occurred. Please try again.")
		}
	}
}

func HandleConfirmation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	if customID == "cancel_remove" {
		respondToInteraction(s, i, "Account removal cancelled.")
		return
	}

	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "confirm_remove_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your confirmation. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to remove it.")
		return
	}

	tx := database.DB.Begin()

	if err := tx.Where("account_id = ?", account.ID).Delete(&models.Ban{}).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Error deleting associated bans")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	if err := tx.Delete(&account).Error; err != nil {
		tx.Rollback()
		logger.Log.WithError(err).Error("Error deleting account")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	if err := tx.Commit().Error; err != nil {
		logger.Log.WithError(err).Error("Error committing transaction")
		respondToInteraction(s, i, "Error removing account. Please try again.")
		return
	}

	respondToInteraction(s, i, fmt.Sprintf("Account '%s' has been successfully removed from the database.", account.Title))
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.UpdateMessageWithPreference(s, i, message, nil, []discordgo.MessageComponent{})

	if err != nil {
		logger.Log.WithError(err).Error("Error updating message, trying to send new message")

		err = services.RespondWithPreference(s, i, message, nil, false)

		if err != nil {
			logger.Log.WithError(err).Error("Error sending new message, trying followup")

			_, err = services.FollowupWithPreference(s, i, message, nil, nil, false)

			if err != nil {
				logger.Log.WithError(err).Error("All response methods failed")
			}
		}
	}
}
