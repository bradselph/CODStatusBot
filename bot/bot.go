package bot

import (
	"errors"
	"os"
	"strings"

	"github.com/bradselph/CODStatusBot/command"
	"github.com/bradselph/CODStatusBot/command/accountage"
	"github.com/bradselph/CODStatusBot/command/accountlogs"
	"github.com/bradselph/CODStatusBot/command/addaccount"
	"github.com/bradselph/CODStatusBot/command/checknow"
	"github.com/bradselph/CODStatusBot/command/feedback"
	"github.com/bradselph/CODStatusBot/command/removeaccount"
	"github.com/bradselph/CODStatusBot/command/setcaptchaservice"
	"github.com/bradselph/CODStatusBot/command/setcheckinterval"
	"github.com/bradselph/CODStatusBot/command/togglecheck"

	"github.com/bradselph/CODStatusBot/command/updateaccount"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bwmarrin/discordgo"
)

var discord *discordgo.Session

func StartBot() (*discordgo.Session, error) {
	envToken := os.Getenv("DISCORD_TOKEN")
	if envToken == "" {
		err := errors.New("DISCORD_TOKEN environment variable not set")
		logger.Log.WithError(err).WithField("env", "DISCORD_TOKEN").Error()
		return nil, err
	}

	var err error
	discord, err = discordgo.New("Bot " + envToken)
	if err != nil {
		return nil, err
	}

	err = discord.Open()
	if err != nil {
		return nil, err
	}

	err = discord.UpdateWatchStatus(0, "the Status of your Accounts so you dont have to.")
	if err != nil {
		return nil, err
	}

	command.RegisterCommands(discord)
	logger.Log.Info("Registering global commands")

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			command.HandleCommand(s, i)
		case discordgo.InteractionModalSubmit:
			handleModalSubmit(s, i)
		case discordgo.InteractionMessageComponent:
			handleMessageComponent(s, i)
		}
	})

	return discord, nil
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	switch {
	case customID == "set_captcha_service_modal":
		setcaptchaservice.HandleModalSubmit(s, i)
	case customID == "add_account_modal":
		addaccount.HandleModalSubmit(s, i)
	case strings.HasPrefix(customID, "update_account_modal_"):
		updateaccount.HandleModalSubmit(s, i)
	case customID == "set_check_interval_modal":
		setcheckinterval.HandleModalSubmit(s, i)
	default:
		logger.Log.WithField("customID", customID).Error("Unknown modal submission")
	}
}

func handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	switch {
	case strings.HasPrefix(customID, "feedback_"):
		feedback.HandleFeedbackChoice(s, i)
	case strings.HasPrefix(customID, "account_age_"):
		accountage.HandleAccountSelection(s, i)
	case strings.HasPrefix(customID, "account_logs_"):
		accountlogs.HandleAccountSelection(s, i)
	case customID == "account_logs_all":
		accountlogs.HandleAccountSelection(s, i)
	case strings.HasPrefix(customID, "update_account_"):
		updateaccount.HandleAccountSelection(s, i)
	case strings.HasPrefix(customID, "remove_account_"):
		removeaccount.HandleAccountSelection(s, i)
	case customID == "cancel_remove" || strings.HasPrefix(customID, "confirm_remove_"):
		removeaccount.HandleConfirmation(s, i)
	case strings.HasPrefix(customID, "check_now_"):
		checknow.HandleAccountSelection(s, i)
	case strings.HasPrefix(customID, "toggle_check_"):
		togglecheck.HandleAccountSelection(s, i)
	case strings.HasPrefix(customID, "confirm_reenable_") || customID == "cancel_reenable":
		togglecheck.HandleConfirmation(s, i)
	default:
		logger.Log.WithField("customID", customID).Error("Unknown message component interaction")
	}

}
