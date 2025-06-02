package bot

import (
	"errors"
	"strings"

	"github.com/bradselph/CODStatusBot/command"
	"github.com/bradselph/CODStatusBot/command/accountage"
	"github.com/bradselph/CODStatusBot/command/accountlogs"
	"github.com/bradselph/CODStatusBot/command/addaccount"
	"github.com/bradselph/CODStatusBot/command/checknow"
	"github.com/bradselph/CODStatusBot/command/feedback"
	"github.com/bradselph/CODStatusBot/command/globalannouncement"
	"github.com/bradselph/CODStatusBot/command/listaccounts"
	"github.com/bradselph/CODStatusBot/command/removeaccount"
	"github.com/bradselph/CODStatusBot/command/setcaptchaservice"
	"github.com/bradselph/CODStatusBot/command/setcheckinterval"
	"github.com/bradselph/CODStatusBot/command/setephemeral"
	"github.com/bradselph/CODStatusBot/command/setnotifications"
	"github.com/bradselph/CODStatusBot/command/togglecheck"
	"github.com/bradselph/CODStatusBot/command/updateaccount"
	"github.com/bradselph/CODStatusBot/command/verdansk"
	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

const StatusMessage = "the Status of your Accounts so you dont have to."

var discord *discordgo.Session

func StartBot() (*discordgo.Session, error) {
	cfg := configuration.Get()
	if cfg.Discord.Token == "" {
		return nil, errors.New("discord token not configured")
	}

	var err error
	discord, err = discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return nil, err
	}

	if cfg.Sharding.Enabled {
		discord.ShardID = cfg.Sharding.ShardID
		discord.ShardCount = cfg.Sharding.TotalShards
		logger.Log.Infof("Configured Discord gateway sharding: Shard %d of %d",
			discord.ShardID, discord.ShardCount)
	}

	discord.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsGuilds

	err = discord.Open()
	if err != nil {
		return nil, err
	}

	err = discord.UpdateWatchStatus(0, StatusMessage)
	if err != nil {
		return nil, err
	}

	command.RegisterCommands(discord)
	logger.Log.Info("Registering global commands")

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID != "" {
			appShardManager := services.GetAppShardManager()
			if !appShardManager.GuildBelongsToInstance(i.GuildID) {
				assignedShard := appShardManager.GetGuildShardID(i.GuildID)
				logger.Log.Debugf("Skipping interaction in guild %s (assigned to shard %d, this is shard %d)",
					i.GuildID, assignedShard, appShardManager.ShardID)
				return
			}
		} else {
			userID := getUserIDFromInteraction(i)
			if userID != "" {
				appShardManager := services.GetAppShardManager()
				if !appShardManager.ShardBelongsToInstance(userID) {
					assignedShard := appShardManager.GetUserShardID(userID)
					logger.Log.Debugf("Skipping direct message interaction from user %s (assigned to shard %d, this is shard %d)",
						userID, assignedShard, appShardManager.ShardID)
					return
				}
			}
		}

		installationType := getInstallationType(i)
		logger.Log.Infof("Handling interaction in context: %s", installationType)

		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			command.HandleCommand(s, i)
		case discordgo.InteractionModalSubmit:
			handleModalSubmit(s, i)
		case discordgo.InteractionMessageComponent:
			handleMessageComponent(s, i)
		}
	})

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.GuildID != "" {
			appShardManager := services.GetAppShardManager()
			if !appShardManager.GuildBelongsToInstance(m.GuildID) {
				assignedShard := appShardManager.GetGuildShardID(m.GuildID)
				logger.Log.Debugf("Skipping message in guild %s (assigned to shard %d, this is shard %d)",
					m.GuildID, assignedShard, appShardManager.ShardID)
				return
			}
		} else {
			appShardManager := services.GetAppShardManager()
			if !appShardManager.ShardBelongsToInstance(m.Author.ID) {
				assignedShard := appShardManager.GetUserShardID(m.Author.ID)
				logger.Log.Debugf("Skipping direct message from user %s (assigned to shard %d, this is shard %d)",
					m.Author.ID, assignedShard, appShardManager.ShardID)
				return
			}
		}

		channel, err := s.Channel(m.ChannelID)
		if err == nil && channel.Type == discordgo.ChannelTypeDM {
			logger.Log.Infof("Received DM from user %s (assigned to this shard): %s", m.Author.Username, m.Content)
		}
	})

	return discord, nil
}

func getInstallationType(i *discordgo.InteractionCreate) string {
	if i.GuildID != "" {
		return "server"
	}
	return "direct"
}

func getUserIDFromInteraction(i *discordgo.InteractionCreate) string {
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	return userID
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	switch {
	case strings.HasPrefix(customID, "set_notifications_modal_"):
		setnotifications.HandleModalSubmit(s, i)
	case customID == "set_captcha_service_modal" ||
		strings.HasPrefix(customID, "set_captcha_service_modal_capsolver") ||
		strings.HasPrefix(customID, "set_captcha_service_modal_ezcaptcha") ||
		strings.HasPrefix(customID, "set_captcha_service_modal_2captcha"):
		setcaptchaservice.HandleModalSubmit(s, i)
	case customID == "add_account_modal":
		addaccount.HandleModalSubmit(s, i)
	case strings.HasPrefix(customID, "update_account_modal_"):
		updateaccount.HandleModalSubmit(s, i)
	case customID == "set_check_interval_modal":
		setcheckinterval.HandleModalSubmit(s, i)
	case customID == "global_announcement_modal":
		globalannouncement.HandleModalSubmit(s, i)
	case customID == "verdansk_activision_id_modal":
		verdansk.HandleActivisionIDModal(s, i)
	default:
		logger.Log.WithField("customID", customID).Error("Unknown modal submission")
	}
}

func handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	switch {
	case customID == "listaccounts":
		listaccounts.CommandListAccounts(s, i)
	case strings.HasPrefix(customID, "set_captcha_"):
		setcaptchaservice.HandleCaptchaServiceSelection(s, i)
	case strings.HasPrefix(customID, "feedback_"):
		feedback.HandleFeedbackChoice(s, i)
	case strings.HasPrefix(customID, "set_ephemeral_"):
		setephemeral.HandleEphemeralSelection(s, i)
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
	case customID == "show_interval_modal":
		setcheckinterval.HandleButton(s, i)
	case customID == "verdansk_provide_id" || customID == "verdansk_select_account":
		verdansk.HandleMethodSelection(s, i)
	case strings.HasPrefix(customID, "verdansk_account_"):
		verdansk.HandleAccountSelection(s, i)
	default:
		logger.Log.WithField("customID", customID).Error("Unknown message component interaction")
	}
}
