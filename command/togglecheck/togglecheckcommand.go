package togglecheck

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"

	"github.com/bwmarrin/discordgo"
)

func CommandToggleCheck(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		respondToInteraction(s, i, "You don't have any monitored accounts.")
		return
	}

	// Create buttons for each account
	var components []discordgo.MessageComponent
	var currentRow []discordgo.MessageComponent

	for _, account := range accounts {
		label := fmt.Sprintf("%s (%s)", account.Title, getCheckStatus(account.IsCheckDisabled))
		currentRow = append(currentRow, discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("toggle_check_%d", account.ID),
		})

		if len(currentRow) == 5 {
			components = append(components, discordgo.ActionsRow{Components: currentRow})
			currentRow = []discordgo.MessageComponent{}
		}
	}

	// Add the last row if it is not empty.
	if len(currentRow) > 0 {
		components = append(components, discordgo.ActionsRow{Components: currentRow})
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Select an account to toggle auto check On/Off:",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.User == nil {
		logger.Log.Error("Interaction member or user is nil")
		respondToInteraction(s, i, "An error occurred. Please try again.")
		return
	}

	customID := i.MessageComponentData().CustomID
	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "toggle_check_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to modify it.")
		return
	}

	if account.UserID != i.Member.User.ID {
		respondToInteraction(s, i, "You don't have permission to modify this account.")
		return
	}

	if account.IsCheckDisabled {
		account.LastNotification = time.Now().Unix()
		account.LastCheck = 0
		account.IsCheckDisabled = false
		account.DisabledReason = ""
		message := fmt.Sprintf("Checks for account '%s' have been re-enabled.", account.Title)
		respondToInteraction(s, i, message)
	} else {
		account.IsCheckDisabled = true
		account.DisabledReason = "Manually disabled by user"
		message := fmt.Sprintf("Checks for account '%s' have been disabled.", account.Title)
		respondToInteraction(s, i, message)
	}
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to update account after toggling check")
		respondToInteraction(s, i, "Error toggling account checks. Please try again.")
		return
	}
}

func HandleConfirmation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.User == nil {
		logger.Log.Error("Interaction member or user is nil")
		respondToInteraction(s, i, "An error occurred. Please try again.")
		return
	}

	customID := i.MessageComponentData().CustomID

	if customID == "cancel_reenable" {
		respondToInteraction(s, i, "Re-enabling cancelled.")
		return
	}

	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "confirm_reenable_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your confirmation. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to modify it.")
		return
	}

	if account.UserID != i.Member.User.ID {
		respondToInteraction(s, i, "You don't have permission to modify this account.")
		return
	}

	account.IsCheckDisabled = false
	account.DisabledReason = ""
	account.ConsecutiveErrors = 0
	if err := database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving account changes")
		respondToInteraction(s, i, "Error re-enabling account checks. Please try again.")
		return
	}

	respondToInteraction(s, i, fmt.Sprintf("Checks for account '%s' have been re-enabled.", account.Title))
}

func showConfirmationButtons(s *discordgo.Session, i *discordgo.InteractionCreate, accountID uint, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Confirm Re-enable",
							Style:    discordgo.SuccessButton,
							CustomID: fmt.Sprintf("confirm_reenable_%d", accountID),
						},
						discordgo.Button{
							Label:    "Cancel",
							Style:    discordgo.DangerButton,
							CustomID: "cancel_reenable",
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error showing confirmation buttons")
		respondToInteraction(s, i, "An error occurred. Please try again.")
	}
}

func getCheckStatus(isDisabled bool) string {
	if isDisabled {
		return "disabled"
	}
	return "enabled"
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    message,
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
