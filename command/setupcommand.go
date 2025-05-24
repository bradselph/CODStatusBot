package command

import (
	"time"

	"github.com/bradselph/CODStatusBot/command/accountage"
	"github.com/bradselph/CODStatusBot/command/accountlogs"
	"github.com/bradselph/CODStatusBot/command/addaccount"
	"github.com/bradselph/CODStatusBot/command/checkcaptchabalance"
	"github.com/bradselph/CODStatusBot/command/checknow"
	"github.com/bradselph/CODStatusBot/command/feedback"
	"github.com/bradselph/CODStatusBot/command/globalannouncement"
	"github.com/bradselph/CODStatusBot/command/helpapi"
	"github.com/bradselph/CODStatusBot/command/helpcookie"
	"github.com/bradselph/CODStatusBot/command/listaccounts"
	"github.com/bradselph/CODStatusBot/command/removeaccount"
	"github.com/bradselph/CODStatusBot/command/setcaptchaservice"
	"github.com/bradselph/CODStatusBot/command/setcheckinterval"
	"github.com/bradselph/CODStatusBot/command/setephemeral"
	"github.com/bradselph/CODStatusBot/command/setnotifications"
	"github.com/bradselph/CODStatusBot/command/togglecheck"
	"github.com/bradselph/CODStatusBot/command/updateaccount"
	"github.com/bradselph/CODStatusBot/command/verdansk"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

var Handlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){}

func RegisterCommands(s *discordgo.Session) error {
	logger.Log.Info("Registering global commands")

	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     "globalannouncement",
			Description:              "Send a global announcement to all users (Admin only)",
			DMPermission:             BoolPtr(true),
			DefaultMemberPermissions: Int64Ptr(int64(discordgo.PermissionAdministrator)),
		},
		{
			Name:         "setcaptchaservice",
			Description:  "Set your Captcha service provider and API key (EZCaptcha/2Captcha)",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "setcheckinterval",
			Description:  "Set check interval, notification interval, and notification type",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "setnotifications",
			Description:  "Set your notification preferences (channel or DM)",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "setephemeral",
			Description:  "Toggle ephemeral (only visible to you) responses",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "addaccount",
			Description:  "Add a new account to monitor",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "checkcaptchabalance",
			Description:  "Check your captcha service balance",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "helpapi",
			DMPermission: BoolPtr(true),
			Description:  "Get help on using the bot and setting up your API key",
		},
		{
			Name:         "helpcookie",
			Description:  "Simple guide to getting your SSOCookie",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "accountage",
			Description:  "Check the age and VIP status of an account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "accountlogs",
			Description:  "View the status logs for an account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "checknow",
			Description:  "Check account status now (rate limited for default API key)",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "listaccounts",
			Description:  "List all your monitored accounts with status and last checked time",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "removeaccount",
			Description:  "Remove a monitored account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "updateaccount",
			Description:  "Update a monitored account's information",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "feedback",
			Description:  "Send anonymous feedback to the bot developer",
			DMPermission: BoolPtr(true),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "Your feedback or suggestion",
					Required:    true,
				},
			},
		},
		{
			Name:         "togglecheck",
			Description:  "Toggle checks on/off for a monitored account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "verdansk",
			Description:  "Get your Verdansk Replay stats and images",
			DMPermission: BoolPtr(true),
		},
	}

	Handlers["set_captcha_service_modal_capsolver"] = setcaptchaservice.HandleModalSubmit
	Handlers["set_captcha_service_modal_ezcaptcha"] = setcaptchaservice.HandleModalSubmit
	Handlers["set_captcha_service_modal_2captcha"] = setcaptchaservice.HandleModalSubmit

	Handlers["set_captcha_capsolver"] = setcaptchaservice.HandleCaptchaServiceSelection
	Handlers["set_captcha_ezcaptcha"] = setcaptchaservice.HandleCaptchaServiceSelection
	Handlers["set_captcha_2captcha"] = setcaptchaservice.HandleCaptchaServiceSelection
	Handlers["set_captcha_remove"] = setcaptchaservice.HandleCaptchaServiceSelection

	Handlers["checkcaptchabalance"] = checkcaptchabalance.CommandCheckCaptchaBalance
	Handlers["globalannouncement"] = globalannouncement.CommandGlobalAnnouncement
	Handlers["setcaptchaservice"] = setcaptchaservice.CommandSetCaptchaService
	Handlers["setcheckinterval"] = setcheckinterval.CommandSetCheckInterval
	Handlers["addaccount"] = addaccount.CommandAddAccount
	Handlers["helpcookie"] = helpcookie.CommandHelpCookie
	Handlers["helpapi"] = helpapi.CommandHelpApi
	Handlers["feedback"] = feedback.CommandFeedback
	Handlers["accountage"] = accountage.CommandAccountAge
	Handlers["accountlogs"] = accountlogs.CommandAccountLogs
	Handlers["checknow"] = checknow.CommandCheckNow
	Handlers["listaccounts"] = listaccounts.CommandListAccounts
	Handlers["removeaccount"] = removeaccount.CommandRemoveAccount
	Handlers["updateaccount"] = updateaccount.CommandUpdateAccount
	Handlers["togglecheck"] = togglecheck.CommandToggleCheck
	Handlers["setnotifications"] = setnotifications.CommandSetNotifications
	Handlers["setephemeral"] = setephemeral.CommandSetEphemeral
	Handlers["verdansk"] = verdansk.CommandVerdansk

	Handlers["set_notifications_modal"] = setnotifications.HandleModalSubmit
	Handlers["setcaptchaservice_modal"] = setcaptchaservice.HandleModalSubmit
	Handlers["addaccount_modal"] = addaccount.HandleModalSubmit
	Handlers["update_account_modal"] = updateaccount.HandleModalSubmit
	Handlers["set_check_interval_modal"] = setcheckinterval.HandleModalSubmit
	Handlers["verdansk_activision_id_modal"] = verdansk.HandleActivisionIDModal

	Handlers["account_age"] = accountage.HandleAccountSelection
	Handlers["account_logs"] = accountlogs.HandleAccountSelection
	Handlers["remove_account"] = removeaccount.HandleAccountSelection
	Handlers["check_now"] = checknow.HandleAccountSelection
	Handlers["toggle_check"] = togglecheck.HandleAccountSelection
	Handlers["feedback_anonymous"] = feedback.HandleFeedbackChoice
	Handlers["feedback_with_id"] = feedback.HandleFeedbackChoice
	Handlers["set_ephemeral_enable"] = setephemeral.HandleEphemeralSelection
	Handlers["set_ephemeral_disable"] = setephemeral.HandleEphemeralSelection
	Handlers["show_interval_modal"] = setcheckinterval.HandleButton
	Handlers["verdansk_provide_id"] = verdansk.HandleMethodSelection
	Handlers["verdansk_select_account"] = verdansk.HandleMethodSelection
	Handlers["verdansk_account"] = verdansk.HandleAccountSelection

	Handlers["confirm_remove"] = removeaccount.HandleConfirmation
	Handlers["confirm_reenable"] = togglecheck.HandleConfirmation
	Handlers["cancel_reenable"] = togglecheck.HandleConfirmation

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	if err != nil {
		logger.Log.WithError(err).Error("Error registering global commands")
		return err
	}

	logger.Log.Info("Global commands registered and handlers set up")
	return nil
}

func HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	startTime := time.Now()
	var userID string
	var success bool = true
	var errorDetails string
	var commandName string

	if i.ApplicationCommandData().Name != "" {
		commandName = i.ApplicationCommandData().Name
	} else if i.MessageComponentData().CustomID != "" {
		commandName = i.MessageComponentData().CustomID
	}

	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		errorDetails = "Missing user information"
		success = false

		services.LogCommandExecution(commandName, "unknown", "", false,
			time.Since(startTime).Milliseconds(), errorDetails)
		return
	}

	if err := services.TrackUserInteraction(s, i); err != nil {
		logger.Log.WithError(err).Error("Failed to track user interaction context")
	}

	var userSettings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings")
		errorDetails = "Error getting user settings"
		success = false
	} else if !userSettings.HasSeenAnnouncement {
		if err := globalannouncement.SendGlobalAnnouncement(s, userID); err != nil {
			logger.Log.WithError(err).Error("Error sending announcement to user")
			errorDetails = "Error sending announcement"
			success = false
		} else {
			userSettings.HasSeenAnnouncement = true
			if err := database.DB.Save(&userSettings).Error; err != nil {
				logger.Log.WithError(err).Error("Error updating user settings after sending announcement")
				errorDetails = "Error saving user settings"
				success = false
			}
		}
	}

	if h, ok := Handlers[i.ApplicationCommandData().Name]; ok {
		h(s, i)
	} else if h, ok := Handlers[i.MessageComponentData().CustomID]; ok {
		h(s, i)
	} else {
		logger.Log.Warnf("Unhandled interaction: %s", i.Type)
		errorDetails = "Unhandled interaction type"
		success = false
	}

	services.LogCommandExecution(commandName, userID, i.GuildID, success,
		time.Since(startTime).Milliseconds(), errorDetails)
}
func BoolPtr(b bool) *bool {
	return &b
}

func Int64Ptr(i int64) *int64 {
	return &i
}
