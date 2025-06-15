package updateaccount

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"

	"github.com/bwmarrin/discordgo"
)

func CommandUpdateAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending deferred response")
		return
	}

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		sendFollowupMessage(s, i, "An error occurred while processing your request.")
		return
	}

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching user accounts")
		sendFollowupMessage(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		sendFollowupMessage(s, i, "You don't have any monitored accounts to update.")
		return
	}

	var components []discordgo.MessageComponent

	for _, account := range accounts {
		label := account.Title
		if account.IsVIP {
			label += " ⭐"
		}

		if account.IsCheckDisabled {
			label += " (Disabled)"
			button := services.CreateV2Button(label, fmt.Sprintf("update_account_%d", account.ID), discordgo.SecondaryButton)
			components = append(components, button)
		} else {
			button := services.CreateV2Button(label, fmt.Sprintf("update_account_%d", account.ID), discordgo.PrimaryButton)
			components = append(components, button)
		}
	}

	_, err = services.FollowupWithPreference(s, i, "Select an account to update:", nil, components, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup with account selection")
	}
}

func HandleAccountSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	accountID, err := strconv.Atoi(strings.TrimPrefix(customID, "update_account_"))
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		respondToInteraction(s, i, "Error processing your selection. Please try again.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account")
		respondToInteraction(s, i, "Error: Account not found or you don't have permission to update it.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if account.UserID != userID {
		respondToInteraction(s, i, "Error: You don't have permission to update this account.")
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
							Placeholder: "Enter the new SSO cookie for this account",
							Required:    true,
							MinLength:   1,
							MaxLength:   100,
						},
					},
				},
			},
		},
	})

	if err != nil {
		logger.Log.WithError(err).Error("Error showing update modal")
		respondToInteraction(s, i, "Error showing update form. Please try again.")
		return
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending deferred response")
		return
	}

	data := i.ModalSubmitData()
	accountIDStr := strings.TrimPrefix(data.CustomID, "update_account_modal_")
	accountID, err := strconv.Atoi(accountIDStr)
	if err != nil {
		logger.Log.WithError(err).Error("Error parsing account ID")
		sendFollowupMessage(s, i, "Error processing your update. Please try again.")
		return
	}

	var newSSOCookie string
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, rowComp := range row.Components {
				if v, ok := rowComp.(*discordgo.TextInput); ok && v.CustomID == "new_sso_cookie" {
					newSSOCookie = utils.SanitizeInput(strings.TrimSpace(v.Value))
				}
			}
		}
	}

	_, err = services.FollowupWithPreference(s, i, "Processing your account update... This may take a few moments.", nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending processing message")
	}
	go processAccountUpdate(s, i, accountID, newSSOCookie)
}
func processAccountUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, accountID int, newSSOCookie string) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	logger.Log.Infof("Processing account update for ID %d by user %s", accountID, userID)

	if !services.VerifySSOCookie(newSSOCookie) {
		logger.Log.Error("SSO cookie validation failed")
		sendFollowupMessage(s, i, "Error: SSO cookie validation failed. Please check the cookie and try again.")
		return
	}

	validationResult, err := services.ValidateAndGetAccountInfo(newSSOCookie)
	if err != nil {
		logger.Log.WithError(err).Error("Error validating new SSO cookie")
		sendFollowupMessage(s, i, fmt.Sprintf("Error validating cookie: %v", err))
		return
	}

	if !validationResult.IsValid {
		sendFollowupMessage(s, i, "Error: The provided SSO cookie is invalid.")
		return
	}

	var account models.Account
	result := database.DB.First(&account, accountID)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching account for update")
		sendFollowupMessage(s, i, "Error: Account not found or you don't have permission to update it.")
		return
	}
	if account.UserID != userID {
		sendFollowupMessage(s, i, "Error: You don't have permission to update this account.")
		return
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		sendFollowupMessage(s, i, "Error fetching user settings. Please try again.")
		return
	}

	if !services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		msg := fmt.Sprintf("Your preferred captcha service (%s) is currently disabled. ", userSettings.PreferredCaptchaProvider)
		if services.IsServiceEnabled("ezcaptcha") {
			msg += "Please switch to EZCaptcha using /setcaptchaservice."
		} else if services.IsServiceEnabled("2captcha") {
			msg += "Please switch to 2Captcha using /setcaptchaservice."
		}
		sendFollowupMessage(s, i, msg)
		return
	}
	services.DBMutex.Lock()
	oldVIP := account.IsVIP
	wasDisabled := account.IsCheckDisabled

	account.LastNotification = time.Now().Unix()
	account.LastCookieNotification = 0
	account.SSOCookie = newSSOCookie
	account.SSOCookieExpiration = validationResult.ExpiresAt
	account.Created = validationResult.Created
	account.IsVIP = validationResult.IsVIP
	account.IsExpiredCookie = false
	account.IsCheckDisabled = false
	account.DisabledReason = ""
	account.ConsecutiveErrors = 0
	account.LastSuccessfulCheck = time.Now()

	if err := database.DB.Save(&account).Error; err != nil {
		services.DBMutex.Unlock()
		logger.Log.WithError(err).Error("Failed to update account")
		sendFollowupMessage(s, i, "Error updating account. Please try again.")
		return
	}
	services.DBMutex.Unlock()

	statusLog := models.Ban{
		AccountID: account.ID,
		Status:    account.LastStatus,
		LogType:   "cookie_update",
		Message:   "SSO Cookie updated",
		Timestamp: time.Now(),
		Initiator: "user",
	}

	if err := database.DB.Create(&statusLog).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to log cookie update")
	}

	var vipStatusChange string
	newVIP := validationResult.IsVIP
	if oldVIP != newVIP {
		if newVIP {
			vipStatusChange = "Your account is now a VIP account!"
		} else {
			vipStatusChange = "Your account is no longer a VIP account"
		}
	}

	embed := createSuccessEmbed(&account, wasDisabled, vipStatusChange, validationResult.ExpiresAt, account.IsVIP)
	sendFollowupMessageWithEmbed(s, i, "", embed)

	go func() {
		time.Sleep(2 * time.Second)
		logger.Log.Infof("Performing status check for updated account %d", account.ID)

		status, err := services.CheckAccount(newSSOCookie, userID, "")
		if err != nil {
			logger.Log.WithError(err).Error("Error performing status check after update")
			return
		}

		services.HandleStatusChange(s, account, status, &userSettings)
	}()
}

func createSuccessEmbed(account *models.Account, wasDisabled bool, vipStatusChange string, expirationTimestamp int64, isVIP bool) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "Account Update Successful",
		Description: fmt.Sprintf("Account '%s' has been updated successfully.", account.Title),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Cookie Expiration",
				Value:  services.FormatExpirationTime(expirationTimestamp),
				Inline: true,
			},
			{
				Name:   "VIP Status",
				Value:  getVIPStatusText(isVIP),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if wasDisabled {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Account Status",
			Value:  "Account checks have been re-enabled",
			Inline: false,
		})
	}

	if vipStatusChange != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status Change",
			Value:  vipStatusChange,
			Inline: false,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Notification Type",
		Value:  account.NotificationType,
		Inline: true,
	})

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Use /listaccounts to view all your monitored accounts",
	}

	return embed
}

func getVIPStatusText(isVIP bool) string {
	if isVIP {
		return "VIP Account"
	}
	return "Regular Account"
}

func sendFollowupMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	if i.Interaction == nil || i.Interaction.Token == "" {
		logger.Log.Error("Cannot send followup: invalid interaction or token")
		return
	}

	_, err := services.FollowupWithPreference(s, i, message, nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func sendFollowupMessageWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, message string, embed *discordgo.MessageEmbed) {
	if i.Interaction == nil || i.Interaction.Token == "" {
		logger.Log.Error("Cannot send followup with embed: invalid interaction or token")
		return
	}

	_, err := services.FollowupWithPreference(s, i, message, []*discordgo.MessageEmbed{embed}, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message with embed")
	}
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction")
	}
}

func respondToInteractionWithMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := services.RespondWithPreference(s, i, message, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error responding to interaction with message")
	}
}
