package command

import (
	"github.com/bradselph/CODStatusBot/command/accountage"
	"github.com/bradselph/CODStatusBot/command/accountlogs"
	"github.com/bradselph/CODStatusBot/command/addaccount"
	"github.com/bradselph/CODStatusBot/command/checknow"
	"github.com/bradselph/CODStatusBot/command/feedback"
	"github.com/bradselph/CODStatusBot/command/globalannouncement"
	"github.com/bradselph/CODStatusBot/command/helpapi"
	"github.com/bradselph/CODStatusBot/command/helpcookie"
	"github.com/bradselph/CODStatusBot/command/listaccounts"
	"github.com/bradselph/CODStatusBot/command/removeaccount"
	"github.com/bradselph/CODStatusBot/command/setcaptchaservice"
	"github.com/bradselph/CODStatusBot/command/setcheckinterval"
	"github.com/bradselph/CODStatusBot/command/togglecheck"
	"github.com/bradselph/CODStatusBot/command/updateaccount"
	"github.com/bradselph/CODStatusBot/database"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

var Handlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){}

func RegisterCommands(s *discordgo.Session) error {
	logger.Log.Info("Registering global commands")

	commands := []*discordgo.ApplicationCommand{
		{
			Name:         "globalannouncement",
			Description:  "Send a global announcement to all users (Admin only)",
			DMPermission: BoolPtr(false),
		},
		{
			Name:         "setcaptchaservice",
			Description:  "Set your EZ-Captcha API key",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "setcheckinterval",
			Description:  "Set check interval, notification interval, and notification type",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "addaccount",
			Description:  "Add a new account to monitor",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "helpapi",
			Description:  "Get help on using the bot and setting up your API key",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "helpcookie",
			Description:  "Simple guide to getting your SSOCookie",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "accountage",
			Description:  "Check the age of an account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "accountlogs",
			Description:  "View the logs for an account",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "checknow",
			Description:  "Check All accounts Now (rate limited for default API key)",
			DMPermission: BoolPtr(true),
		},
		{
			Name:         "listaccounts",
			Description:  "List all your monitored accounts",
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
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
	if err != nil {
		logger.Log.WithError(err).Error("Error registering global commands")
		return err
	}

	// Command handlers
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

	// Command Modal Handlers
	Handlers["setcaptchaservice_modal"] = setcaptchaservice.HandleModalSubmit
	Handlers["addaccount_modal"] = addaccount.HandleModalSubmit
	Handlers["updateaccount_modal"] = updateaccount.HandleModalSubmit
	Handlers["setcheckinterval_modal"] = setcheckinterval.HandleModalSubmit

	// Command select handlers
	Handlers["account_age"] = accountage.HandleAccountSelection
	Handlers["account_logs"] = accountlogs.HandleAccountSelection
	Handlers["remove_account"] = removeaccount.HandleAccountSelection
	Handlers["check_now"] = checknow.HandleAccountSelection
	Handlers["toggle_check"] = togglecheck.HandleAccountSelection
	Handlers["feedback_anonymous"] = feedback.HandleFeedbackChoice
	Handlers["feedback_with_id"] = feedback.HandleFeedbackChoice

	// Confirmation handlers
	Handlers["confirm_remove"] = removeaccount.HandleConfirmation

	logger.Log.Info("Global commands registered and handlers set up")
	return nil
}

// HandleCommand handles incoming commands and checks for announcements
func HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if the user has seen the announcement
	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		logger.Log.Error("Interaction doesn't have Member or User")
		return
	}

	var userSettings models.UserSettings
	result := database.DB.Where(models.UserSettings{UserID: userID}).FirstOrCreate(&userSettings)
	if result.Error != nil {
		logger.Log.WithError(result.Error).Error("Error getting user settings")
	} else if !userSettings.HasSeenAnnouncement {
		// Send the announcement to the user
		if err := globalannouncement.SendGlobalAnnouncement(s, userID); err != nil {
			logger.Log.WithError(err).Error("Error sending announcement to user")
		} else {
			// Update the user's settings to mark the announcement as seen.
			userSettings.HasSeenAnnouncement = true
			if err := database.DB.Save(&userSettings).Error; err != nil {
				logger.Log.WithError(err).Error("Error updating user settings after sending announcement")
			}
		}
	}

	if h, ok := Handlers[i.ApplicationCommandData().Name]; ok {
		h(s, i)
	} else if h, ok := Handlers[i.MessageComponentData().CustomID]; ok {
		h(s, i)
	} else {
		logger.Log.Warnf("Unhandled interaction: %s", i.Type)
	}
}

// BoolPtr Helper function to create a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}
