package listaccounts

import (
	"fmt"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bradselph/CODStatusBot/utils"
	"github.com/bwmarrin/discordgo"
)

/*
var (
	//	checkCircle    = os.Getenv("CHECKCIRCLE")
	banCircle = os.Getenv("BANCIRCLE")
	//	infoCircle     = os.Getenv("INFOCIRCLE")
	stopWatch      = os.Getenv("STOPWATCH")
	questionCircle = os.Getenv("QUESTIONCIRCLE")
)
*/

func CommandListAccounts(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer response")
		return
	}

	userID, err := services.GetUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Could not determine user ID")
		sendFollowup(s, i, "An error occurred while processing your request.")
		return
	}

	if err := utils.ValidateDiscordUserID(userID); err != nil {
		logger.Log.WithError(err).WithField("userID", userID).Error("Invalid user ID")
		sendFollowup(s, i, "Invalid user ID provided.")
		return
	}

	var accounts []models.Account
	result := database.DB.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error fetching user accounts")
		sendFollowup(s, i, "Error fetching your accounts. Please try again.")
		return
	}

	if len(accounts) == 0 {
		sendFollowup(s, i, "You don't have any monitored accounts.")
		return
	}

	balanceInfo := getBalanceInfo(userID)
	description := "Here's a detailed list of all your monitored accounts:"
	if balanceInfo != "" {
		description += balanceInfo
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Monitored Accounts",
		Description: description,
		Color:       0x00ff00,
		Fields:      make([]*discordgo.MessageEmbedField, 0),
	}

	cfg := configuration.Get()

	for _, account := range accounts {
		checkStatus := services.GetCheckStatus(account.IsCheckDisabled)
		cookieExpiration := services.FormatExpirationTime(account.SSOCookieExpiration)
		creationDate := time.Unix(account.Created, 0).Format("2006-01-02")
		lastCheckTime := time.Unix(account.LastCheck, 0).Format("2006-01-02 15:04:05")

		vipStatus := "No"
		if account.IsVIP {
			vipStatus = "Yes ✓"
		}

		ogVerdan := "No"
		if account.IsOGVerdansk {
			ogVerdan = "OG Verdansk"
		}

		fieldValue := fmt.Sprintf("Status: %s\n", account.LastStatus)

		if account.IsPermabanned {
			fieldValue += cfg.Emojis.BanCircle + "Account Permanently Banned\n"
		}
		if account.IsTempbanned {
			fieldValue += cfg.Emojis.StopWatch + "Account Temporarily Banned\n"
		}
		if account.IsShadowbanned {
			fieldValue += cfg.Emojis.QuestionCircle + "Account Under Review\n"
		}
		if account.IsExpiredCookie {
			fieldValue += "⚠ Cookie Expired\n"
		}
		if account.ConsecutiveErrors > 0 {
			fieldValue += fmt.Sprintf("⚠ Check Errors: %d\n", account.ConsecutiveErrors)
		}

		fieldValue += fmt.Sprintf("VIP Status: %s\nOG Verdansk: %s\nChecks: %s\nNotification Type: %s\n"+
			"Cookie Expires: %s\nCreated: %s\nLast Checked: %s",
			vipStatus, ogVerdan, checkStatus, account.NotificationType,
			cookieExpiration, creationDate, lastCheckTime)

		if account.IsCheckDisabled {
			fieldValue += fmt.Sprintf("\nDisabled Reason: %s", account.DisabledReason)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", account.Title, getDisabledEmoji(account.IsCheckDisabled)),
			Value:  fieldValue,
			Inline: false,
		})

		color := services.GetColorForStatus(account.LastStatus, account.IsExpiredCookie, account.IsCheckDisabled)
		if color != 0x00ff00 {
			embed.Color = color
		}
	}

	embed = services.ValidateEmbedLimits(embed)

	_, err = services.FollowupWithPreference(s, i, "", []*discordgo.MessageEmbed{embed}, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func sendFollowup(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := services.FollowupWithPreference(s, i, content, nil, nil, false)
	if err != nil {
		logger.Log.WithError(err).Error("Error sending followup message")
	}
}

func getDisabledEmoji(isDisabled bool) string {
	if isDisabled {
		return "⛔"
	}
	return "✓"
}

func getBalanceInfo(userID string) string {
	userSettings, err := services.GetUserSettings(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error fetching user settings")
		return ""
	}

	if !services.IsServiceEnabled(userSettings.PreferredCaptchaProvider) {
		return ""
	}

	hasCustomKey := userSettings.CapSolverAPIKey != "" ||
		userSettings.EZCaptchaAPIKey != "" ||
		userSettings.TwoCaptchaAPIKey != ""

	if !hasCustomKey {
		return "\n\nYou are using the bot's default API key. Consider setting up your own key using /setcaptchaservice for unlimited checks."
	}

	var balance float64
	var providerName string

	switch userSettings.PreferredCaptchaProvider {
	case "capsolver":
		if userSettings.CapSolverAPIKey != "" {
			_, balance, err = services.ValidateCaptchaKey(userSettings.CapSolverAPIKey, "capsolver")
			providerName = "Capsolver"
		}
	case "ezcaptcha":
		if userSettings.EZCaptchaAPIKey != "" {
			_, balance, err = services.ValidateCaptchaKey(userSettings.EZCaptchaAPIKey, "ezcaptcha")
			providerName = "EZCaptcha"
		}
	case "2captcha":
		if userSettings.TwoCaptchaAPIKey != "" {
			_, balance, err = services.ValidateCaptchaKey(userSettings.TwoCaptchaAPIKey, "2captcha")
			providerName = "2Captcha"
		}
	}

	if err != nil {
		logger.Log.WithError(err).Error("Error validating user captcha key")
		return "\n\nError retrieving balance information."
	}

	var threshold float64
	switch userSettings.PreferredCaptchaProvider {
	case "ezcaptcha":
		threshold = 250
	case "2captcha":
		threshold = 0.25
	default:
		threshold = 250
	}

	balanceMsg := fmt.Sprintf("\n\nYour current %s balance: %.2f points",
		providerName, balance)

	if balance < threshold {
		balanceMsg += fmt.Sprintf(" (Warning: Below recommended %.2f points)", threshold)
	}

	return balanceMsg
}
