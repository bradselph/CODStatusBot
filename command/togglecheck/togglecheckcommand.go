package togglecheck

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

func CommandToggleCheck(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
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

	var components []discordgo.MessageComponent

	for _, account := range accounts {
		label := fmt.Sprintf("%s (%s)", account.Title, services.GetCheckStatus(account.IsCheckDisabled))
		components = append(components, discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("toggle_check_%d", account.ID),
		})
	}

	err = services.RespondWithPreferenceAndComponents(s, i, "Select an account to toggle auto check On/Off:", nil, components, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error acknowledging interaction")
		return
	}

	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		sendFollowupMessage(s, i, "An error occurred while processing your request.")
		return
	}

	customID := i.MessageComponentData().CustomID
	accountIDParsed, err := strconv.ParseUint(strings.TrimPrefix(customID, "toggle_check_"), 10, 64)
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		sendFollowupMessage(s, i, "Error processing your selection. Please try again.")
		return
	}
	accountID := uint(accountIDParsed)

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		sendFollowupMessage(s, i, "Error: Account not found or you don't have permission to modify it.")
		return
	}

	if account.UserID != userID {
		sendFollowupMessage(s, i, "You don't have permission to modify this account.")
		return
	}

	if account.IsCheckDisabled {
		showConfirmationButtons(s, i, accountID, fmt.Sprintf("Are you sure you want to re-enable checks for account '%s'?", account.Title))
	} else {
		account.IsCheckDisabled = true
		account.DisabledReason = "Manually disabled by user"
		message := fmt.Sprintf("Checks for account '%s' have been disabled.", account.Title)
		if err = database.DB.Save(&account).Error; err != nil {
			logger.Log.WithError(err).Error("Failed to update account after toggling check")
			sendFollowupMessage(s, i, "Error toggling account checks. Please try again.")
			return
		}
		sendFollowupMessage(s, i, message)
	}
}

func showConfirmationButtons(s *discordgo.Session, i *discordgo.InteractionCreate, accountID uint, message string) {
	logger.Log.Infof("Showing confirmation buttons for account %d", accountID)

	confirmComponents := []discordgo.MessageComponent{
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
	}

	_, err := services.FollowupWithPreference(s, i, message, nil, confirmComponents, false)

	if err != nil {
		logger.Log.WithError(err).Error("Error showing confirmation buttons")
		sendFollowupMessage(s, i, "An error occurred. Please try again.")
		return
	}
}

func sendFollowupMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	_, err := services.FollowupWithPreference(s, i, message, nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func HandleConfirmation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer interaction response")
		return
	}

	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		sendFollowupMessage(s, i, "An error occurred while processing your request.")
		return
	}

	customID := i.MessageComponentData().CustomID

	if customID == "cancel_reenable" {
		sendFollowupMessage(s, i, "Re-enabling cancelled.")
		return
	}

	accountIDParsed, err := strconv.ParseUint(strings.TrimPrefix(customID, "confirm_reenable_"), 10, 64)
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		sendFollowupMessage(s, i, "Error processing your confirmation. Please try again.")
		return
	}
	accountID := uint(accountIDParsed)

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		sendFollowupMessage(s, i, "Error: Account not found or you don't have permission to modify it.")
		return
	}

	if account.UserID != userID {
		sendFollowupMessage(s, i, "You don't have permission to modify this account.")
		return
	}

	cookieValid := services.VerifySSOCookie(account.SSOCookie)
	if !cookieValid {
		logger.Log.Warnf("Re-enabling account with potentially expired cookie: %s (ID: %d)", account.Title, account.ID)
		account.IsExpiredCookie = true
	} else {
		account.IsExpiredCookie = false
	}

	account.IsCheckDisabled = false
	account.DisabledReason = ""
	account.ConsecutiveErrors = 0
	if err = database.DB.Save(&account).Error; err != nil {
		logger.Log.WithError(err).Error("Error saving account changes")
		sendFollowupMessage(s, i, "Error re-enabling account checks. Please try again.")
		return
	}

	message := fmt.Sprintf("Checks for account '%s' have been re-enabled.", account.Title)
	if !cookieValid {
		message += "\n Note: The account's SSO cookie may be expired. If checks fail, please update the cookie using /updateaccount."
	}

	sendFollowupMessage(s, i, message)
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.UpdateMessageWithPreference(s, i, message, nil, []discordgo.MessageComponent{})

	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}
