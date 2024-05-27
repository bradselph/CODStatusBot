package command

import (
	"codstatusbot/command/accountage"
	"codstatusbot/command/accountlogs"
	"codstatusbot/command/addaccount"
	"codstatusbot/command/help"
	"codstatusbot/command/removeaccount"
	"codstatusbot/command/updateaccount"
	"codstatusbot/logger"

	"github.com/bwmarrin/discordgo"
)

var CommandHandlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){}

// RegisterCommand registers all command handlers for a specific guild.
func RegisterCommand(s *discordgo.Session, guildID string) {
	logger.Log.Info("Registering commands by command handler")
	addaccount.RegisterCommand(s, guildID)
	logger.Log.Info("Registering addaccount command")
	CommandHandlers["addaccount"] = addaccount.CommandAddAccount
	removeaccount.RegisterCommand(s, guildID)
	logger.Log.Info("Registering removeaccount command")
	CommandHandlers["removeaccount"] = removeaccount.CommandRemoveAccount
	accountlogs.RegisterCommand(s, guildID)
	logger.Log.Info("Registering accountlogs command")
	CommandHandlers["accountlogs"] = accountlogs.CommandAccountLogs
	updateaccount.RegisterCommand(s, guildID)
	logger.Log.Info("Registering updateaccount command")
	CommandHandlers["updateaccount"] = updateaccount.CommandUpdateAccount
	accountage.RegisterCommand(s, guildID)
	logger.Log.Info("Registering accountage command")
	CommandHandlers["accountage"] = accountage.CommandAccountAge
	help.RegisterCommand(s, guildID)
	logger.Log.Info("Registering help command")
	CommandHandlers["help"] = help.CommandHelp
}

// UnregisterCommand unregisters all command handlers for a specific guild.
func UnregisterCommand(s *discordgo.Session, guildID string) {
	logger.Log.Info("Unregistering commands by command handler")
	addaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering addaccount command")
	removeaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering removeaccount command")
	accountlogs.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering accountlogs command")
	updateaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering updateaccount command")
	accountage.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering accountage command")
	help.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering help command")
}