package listaccounts

import (
	"fmt"
	"os"
	"time"

	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

var (
	//	checkCircle    = os.Getenv("CHECKCIRCLE")
	banCircle = os.Getenv("BANCIRCLE")
	//	infoCircle     = os.Getenv("INFOCIRCLE")
	stopWatch      = os.Getenv("STOPWATCH")
	questionCircle = os.Getenv("QUESTIONCIRCLE")
)

func CommandListAccounts(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := services.DeferWithPreference(s, i, false)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to defer response")
		return
	}

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		sendFollowup(s, i, "An error occurred while processing your request.")
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
			fieldValue += banCircle + "Account Permanently Banned\n"
		}
		if account.IsTempbanned {
			fieldValue += stopWatch + "Account Temporarily Banned\n"
		}
		if account.IsShadowbanned {
			fieldValue += questionCircle + "Account Under Review\n"
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

	apiKey, balance, err := services.GetUserCaptchaKey(userID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting user captcha key")
		return ""
	}

	if apiKey == "" {
		return "\n\nYou are using the bot's default API key. Consider setting up your own key using /setcaptchaservice for unlimited checks."
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
		userSettings.PreferredCaptchaProvider, balance)

	if balance < threshold {
		balanceMsg += fmt.Sprintf(" (Warning: Below recommended %.2f points)", threshold)
	}

	return balanceMsg
}
