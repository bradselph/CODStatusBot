package addaccount

import (
	"fmt"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var (
	rateLimit time.Duration
)

func init() {
	cfg := configuration.Get()
	rateLimit = cfg.RateLimits.CheckNow
}

func getMaxAccounts(hasCustomKey bool) int {
	cfg := configuration.Get()
	if hasCustomKey {
		return cfg.RateLimits.PremiumMaxAccounts
	}
	return cfg.RateLimits.DefaultMaxAccounts
}

func CommandAddAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID, err := services.GetUserID(i)
	if userID == "" {
		logger.Log.Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		respondToInteraction(s, i, "Error fetching user settings. Please try again.")
		return
	}

	hasCustomKey := userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != ""
	if !hasCustomKey && !checkRateLimit(userID) {
		respondToInteraction(s, i, fmt.Sprintf("Please wait %v before adding another account.", rateLimit))
		return
	}

	if !services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		msg := fmt.Sprintf("Your preferred captcha service (%s) is currently disabled. ", userSettings.PreferredCaptchaProvider)
		if services.IsServiceEnabled("ezcaptcha") {
			msg += "Please set up EZCaptcha using /setcaptchaservice before adding accounts."
		} else if services.IsServiceEnabled("2captcha") {
			msg += "Please set up 2Captcha using /setcaptchaservice before adding accounts."
		} else if services.IsServiceEnabled("capsolver") {
			msg += "Please set up Capsolver using /setcaptchaservice before adding accounts."
		} else {
			msg += "No captcha services are currently available. Please try again later."
		}
		respondToInteraction(s, i, msg)
		return
	}

	if userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != "" {
		_, balance, err := services.GetUserCaptchaKey(userID)
		if err != nil {
			logger.Log.WithError(err).Error("Error checking captcha balance")
			respondToInteraction(s, i, "Error validating your captcha API key. Please check your key using /setcaptchaservice.")
			return
		}

		if balance <= 0 {
			respondToInteraction(s, i, fmt.Sprintf("Your captcha balance (%.2f) is too low to add new accounts. Please recharge your balance.", balance))
			return
		}
	}

	var accountCount int64
	if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
		logger.Log.WithError(err).Error("Error counting user accounts")
		respondToInteraction(s, i, "Error checking account limit. Please try again.")
		return
	}

	maxAccounts := getMaxAccounts(hasCustomKey)

	if accountCount >= int64(maxAccounts) {
		msg := fmt.Sprintf("You've reached the maximum limit of %d accounts.", maxAccounts)
		if !hasCustomKey {
			msg += " Upgrade to premium by adding your own API key using /setcaptchaservice to increase your account limit!"
		} else {
			msg += " Please remove some accounts before adding new ones."
		}
		respondToInteraction(s, i, msg)
		return
	}

	showAddAccountModal(s, i)
}

func showAddAccountModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
							Placeholder: "Enter a name for this account",
							Required:    true,
							MinLength:   3,
							MaxLength:   40,
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
							MinLength:   60,
							MaxLength:   95,
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error showing add account modal")
	}
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID, _ := services.GetUserID(i)

	if userID == "" {
		logger.Log.Error("Failed to get user ID")
		respondToInteraction(s, i, "An error occurred while processing your request.")
		return
	}

	title := utils.SanitizeInput(strings.TrimSpace(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))
	ssoCookie := strings.TrimSpace(data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

	logger.Log.Infof("Attempting to add account. Title: %s, SSO Cookie length: %d", title, len(ssoCookie))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error sending deferred response")
		return
	}

	validationResult, err := services.ValidateAndGetAccountInfo(ssoCookie)
	if err != nil {
		logger.Log.WithError(err).Error("Error validating account")
		sendFollowupMessage(s, i, fmt.Sprintf("Error processing account: %v", err))
		return
	}

	if !validationResult.IsValid {
		logger.Log.Error("Invalid SSO cookie provided")
		sendFollowupMessage(s, i, "Invalid SSO cookie. Please make sure you've copied the entire cookie value.")
		return
	}

	channelID := getChannelID(s, i)
	if channelID == "" {
		sendFollowupMessage(s, i, "An error occurred while processing your request.")
		return
	}

	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		sendFollowupMessage(s, i, "Error fetching user settings. Please try again.")
		return
	}

	var accountCount int64
	if err := database.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&accountCount).Error; err != nil {
		logger.Log.WithError(err).Error("Error counting user accounts")
		sendFollowupMessage(s, i, "Error checking account limit. Please try again.")
		return
	}

	hasCustomKey := userSettings.CapSolverAPIKey != "" || userSettings.EZCaptchaAPIKey != "" || userSettings.TwoCaptchaAPIKey != ""
	maxAccounts := getMaxAccounts(hasCustomKey)

	if accountCount >= int64(maxAccounts) {
		msg := fmt.Sprintf("You've reached the maximum limit of %d accounts.", maxAccounts)
		if !hasCustomKey {
			msg += " Upgrade to premium by adding your own API key using /setcaptchaservice to increase your account limit!"
		} else {
			msg += " Please remove some accounts before adding new ones."
		}
		sendFollowupMessage(s, i, msg)
		return
	}

	account := models.Account{
		UserID:              userID,
		Title:               title,
		SSOCookie:           ssoCookie,
		SSOCookieExpiration: validationResult.ExpiresAt,
		Created:             validationResult.Created,
		IsVIP:               validationResult.IsVIP,
		ChannelID:           channelID,
		NotificationType:    userSettings.NotificationType,
		LastSuccessfulCheck: time.Now(),
		LastStatus:          models.StatusUnknown,
	}

	if err := database.DB.Create(&account).Error; err != nil {
		logger.Log.WithError(err).WithFields(logrus.Fields{"userID": userID, "title": title}).Error("Error creating account")
		sendFollowupMessage(s, i, "Error creating account. Please try again.")
		return
	}

	accountLog := models.Ban{
		AccountID: account.ID,
		Status:    models.StatusUnknown,
		LogType:   "account_added",
		Message:   fmt.Sprintf("Account '%s' was added to monitoring", account.Title),
		Timestamp: time.Now(),
	}

	if err := database.DB.Create(&accountLog).Error; err != nil {
		logger.Log.WithError(err).Error("Failed to create account creation log")
	}

	vipStatus := "Regular Account"
	if account.IsVIP {
		vipStatus = "VIP Account"
	}

	remainingSlots := maxAccounts - int(accountCount) - 1
	slotInfo := fmt.Sprintf("\nYou have %d account slot(s) remaining.", remainingSlots)

	embed := &discordgo.MessageEmbed{
		Title:       "Account Added Successfully",
		Description: fmt.Sprintf("Account '%s' has been added to monitoring.%s", account.Title, slotInfo),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Account Type",
				Value:  vipStatus,
				Inline: true,
			},
			{
				Name:   "Cookie Expiration",
				Value:  services.FormatExpirationTime(account.SSOCookieExpiration),
				Inline: true,
			},
			{
				Name:   "Account Age",
				Value:  formatAccountAge(time.Unix(account.Created, 0)),
				Inline: true,
			},
			{
				Name:   "Notification Type",
				Value:  account.NotificationType,
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /listaccounts to view all your monitored accounts",
		},
	}

	sendFollowupMessageWithEmbed(s, i, "Account added successfully!", embed)

	go func() {
		time.Sleep(2 * time.Second)

		status, err := services.CheckAccount(ssoCookie, userID, "")
		if err != nil {
			logger.Log.WithError(err).Error("Error performing initial status check")
			return
		}

		var updatedAccount models.Account
		if err := database.DB.First(&updatedAccount, account.ID).Error; err != nil {
			logger.Log.WithError(err).Error("Error fetching account for initial status update")
			return
		}

		services.HandleStatusChange(s, updatedAccount, status, &userSettings)
	}()
}

func sendFollowupMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func sendFollowupMessageWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, content string, embed *discordgo.MessageEmbed) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message with embed")
	}
}

func formatAccountAge(created time.Time) string {
	age := time.Since(created)
	years := int(age.Hours() / 24 / 365)
	months := int(age.Hours()/24/30.44) % 12
	return fmt.Sprintf("%d years, %d months", years, months)
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func getChannelID(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	userID := getUserID(i)
	if userID == "" {
		return ""
	}

	if i.ChannelID != "" {
		return i.ChannelID
	}

	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error creating DM channel")
		return ""
	}
	return channel.ID
}

func checkRateLimit(userID string) bool {
	var userSettings models.UserSettings
	if err := database.DB.Where("user_id = ?", userID).First(&userSettings).Error; err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return false
	}

	userSettings.EnsureMapsInitialized()
	now := time.Now()
	lastAddTime := userSettings.LastCommandTimes["add_account"]

	if lastAddTime.IsZero() || time.Since(lastAddTime) >= rateLimit {
		userSettings.LastCommandTimes["add_account"] = now
		if err := database.DB.Save(&userSettings).Error; err != nil {
			logger.Log.WithError(err).Error("Error saving user settings")
			return false
		}
		return true
	}
	return false
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
