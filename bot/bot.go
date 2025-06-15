package bot

import (
	"errors"
	"fmt"
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
		defer services.RecoverFromPanic("interaction_handler")

		if err := services.ValidateInteractionContext(i); err != nil {
			logger.Log.WithError(err).Error("Invalid interaction context")
			return
		}

		appShardManager := services.GetAppShardManager()
		cfg := configuration.Get()

		if cfg.Sharding.Enabled && appShardManager.Initialized && appShardManager.TotalShards > 1 {
			shouldProcess, reason := shouldProcessInteraction(appShardManager, i)
			if !shouldProcess {
				logger.Log.Debugf("Skipping interaction due to shard assignment: %s", reason)
				return
			}
		} else {
			logger.Log.Debug("Processing interaction (sharding disabled or failed)")
		}

		installationType := getInstallationType(i)
		userID := getUserIDFromInteraction(i)

		logger.Log.Infof("Shard %d processing interaction in context: %s for user: %s",
			appShardManager.ShardID, installationType, userID)

		services.LogAnalyticsEvent("interaction_received", userID, i.GuildID, "", "",
			appShardManager.ShardID, appShardManager.InstanceID, map[string]interface{}{
				"interaction_type":  i.Type.String(),
				"command_name":      getCommandName(i),
				"installation_type": installationType,
			})

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

		appShardManager := services.GetAppShardManager()
		cfg := configuration.Get()

		if cfg.Sharding.Enabled && appShardManager.Initialized && appShardManager.TotalShards > 1 {
			shouldProcess, reason := shouldProcessMessage(appShardManager, m)
			if !shouldProcess {
				logger.Log.Debugf("Skipping message due to shard assignment: %s", reason)
				return
			}
		}

		channel, err := s.Channel(m.ChannelID)
		if err == nil && channel.Type == discordgo.ChannelTypeDM {
			logger.Log.Infof("Shard %d received DM from user %s: %s",
				appShardManager.ShardID, m.Author.Username, m.Content)
		}
	})

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		appShardManager := services.GetAppShardManager()
		logger.Log.Infof("Shard %d/%d Discord session ready with %d guilds",
			appShardManager.ShardID, appShardManager.TotalShards, len(r.Guilds))

		services.LogAnalyticsEvent("shard_ready", "", "", "", "",
			appShardManager.ShardID, appShardManager.InstanceID, map[string]interface{}{
				"guild_count": len(r.Guilds),
				"session_id":  r.SessionID,
			})
	})

	discord.AddHandler(func(s *discordgo.Session, d *discordgo.Disconnect) {
		appShardManager := services.GetAppShardManager()
		logger.Log.Warnf("Shard %d/%d disconnected from Discord",
			appShardManager.ShardID, appShardManager.TotalShards)

		services.LogAnalyticsEvent("shard_disconnect", "", "", "", "",
			appShardManager.ShardID, appShardManager.InstanceID, map[string]interface{}{
				"disconnect_reason": "discord_disconnect_event",
			})
	})

	return discord, nil
}

func shouldProcessInteraction(shardManager *services.AppShardManager, i *discordgo.InteractionCreate) (bool, string) {
	if i.GuildID != "" {
		if !shardManager.GuildBelongsToInstance(i.GuildID) {
			assignedShard := shardManager.GetGuildShardID(i.GuildID)
			return false, fmt.Sprintf("guild %s assigned to shard %d, this is shard %d",
				i.GuildID, assignedShard, shardManager.ShardID)
		}
	} else {
		userID := getUserIDFromInteraction(i)
		if userID != "" {
			if !shardManager.ShardBelongsToInstance(userID) {
				assignedShard := shardManager.GetUserShardID(userID)
				return false, fmt.Sprintf("user %s assigned to shard %d, this is shard %d",
					userID, assignedShard, shardManager.ShardID)
			}
		}
	}
	return true, ""
}

func shouldProcessMessage(shardManager *services.AppShardManager, m *discordgo.MessageCreate) (bool, string) {
	if m.GuildID != "" {
		if !shardManager.GuildBelongsToInstance(m.GuildID) {
			assignedShard := shardManager.GetGuildShardID(m.GuildID)
			return false, fmt.Sprintf("guild %s assigned to shard %d, this is shard %d",
				m.GuildID, assignedShard, shardManager.ShardID)
		}
	} else {
		if !shardManager.ShardBelongsToInstance(m.Author.ID) {
			assignedShard := shardManager.GetUserShardID(m.Author.ID)
			return false, fmt.Sprintf("user %s assigned to shard %d, this is shard %d",
				m.Author.ID, assignedShard, shardManager.ShardID)
		}
	}
	return true, ""
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

func getCommandName(i *discordgo.InteractionCreate) string {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		return i.ApplicationCommandData().Name
	case discordgo.InteractionModalSubmit:
		return i.ModalSubmitData().CustomID
	case discordgo.InteractionMessageComponent:
		return i.MessageComponentData().CustomID
	default:
		return "unknown"
	}
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer services.RecoverFromPanic("handleModalSubmit")

	customID := i.ModalSubmitData().CustomID
	userID := getUserIDFromInteraction(i)
	appShardManager := services.GetAppShardManager()

	logger.Log.Debugf("Shard %d handling modal submit %s for user %s",
		appShardManager.ShardID, customID, userID)

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
		if err := services.RespondWithPreference(s, i, "Unknown modal submission. Please try again.", nil, true); err != nil {
			logger.Log.WithError(err).Error("Failed to respond to unknown modal")
		}
	}
}

func handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer services.RecoverFromPanic("handleMessageComponent")

	customID := i.MessageComponentData().CustomID
	userID := getUserIDFromInteraction(i)
	appShardManager := services.GetAppShardManager()

	logger.Log.Debugf("Shard %d handling message component %s for user %s",
		appShardManager.ShardID, customID, userID)

	switch {
	case customID == "listaccounts":
		listaccounts.CommandListAccounts(s, i)
	case strings.HasPrefix(customID, "set_captcha_"):
		setcaptchaservice.HandleCaptchaServiceSelection(s, i)
	case customID == "toggle_fallback_enabled" || strings.HasPrefix(customID, "set_fallback_") || customID == "captcha_main_menu":
		setcaptchaservice.HandleFallbackSettingsInteraction(s, i)
	case customID == "dismiss_fallback_notice" || customID == "set_captcha_from_notice":
		setcaptchaservice.HandleFallbackNoticeInteraction(s, i)
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
		if err := services.RespondWithPreference(s, i, "Unknown interaction. Please try again.", nil, true); err != nil {
			logger.Log.WithError(err).Error("Failed to respond to unknown component")
		}
	}
}
