package command

import (
	"CODStatusBot/command/accountage"
	"CODStatusBot/command/accountlogs"
	"CODStatusBot/command/addaccount"
	"CODStatusBot/command/addaccountnew"
	"CODStatusBot/command/checknow"
	"CODStatusBot/command/feedback"
	"CODStatusBot/command/help"
	"CODStatusBot/command/removeaccount"
	"CODStatusBot/command/updateaccount"
	"CODStatusBot/logger"

	"github.com/bwmarrin/discordgo"
)

var Handlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){}

func RegisterCommands(s *discordgo.Session, guildID string) {
	logger.Log.Info("Registering commands by command handler")

	removeaccount.RegisterCommand(s, guildID)
	Handlers["removeaccount"] = removeaccount.CommandRemoveAccount
	logger.Log.Info("Registering removeaccount command")

	accountlogs.RegisterCommand(s, guildID)
	Handlers["accountlogs"] = accountlogs.CommandAccountLogs
	logger.Log.Info("Registering accountlogs command")

	updateaccount.RegisterCommand(s, guildID)
	Handlers["updateaccount"] = updateaccount.CommandUpdateAccount
	logger.Log.Info("Registering updateaccount command")

	/*
		setpreference.RegisterCommand(s, guildID)
		Handlers["setpreference"] = setpreference.CommandSetPreference
		logger.Log.Info("Registering setpreference command")
	*/

	accountage.RegisterCommand(s, guildID)
	Handlers["accountage"] = accountage.CommandAccountAge
	logger.Log.Info("Registering accountage command")

	addaccount.RegisterCommand(s, guildID)
	Handlers["addaccount"] = addaccount.CommandAddAccount
	logger.Log.Info("Registering addaccount command")

	addaccountnew.RegisterCommand(s, guildID)
	Handlers["addaccountnew"] = addaccountnew.CommandAddAccountNew
	logger.Log.Info("Registering addaccountnew command")

	/*
		claimrewards.RegisterCommand(s, guildID)
		Handlers["claimavailablerewards"] = claimrewards.CommandClaimRewards
		logger.Log.Info("Registering claimavailablerewards command")
	*/

	checknow.RegisterCommand(s, guildID)
	Handlers["checknow"] = checknow.CommandCheckNow
	logger.Log.Info("Registering checknow command")

	help.RegisterCommand(s, guildID)
	Handlers["help"] = help.CommandHelp
	logger.Log.Info("Registering help command")

	// Register global feedback command
	feedback.RegisterCommand(s)
	Handlers["feedback"] = feedback.CommandFeedback
	logger.Log.Info("Registering global feedback command")

}

func UnregisterCommands(s *discordgo.Session, guildID string) {
	logger.Log.Info("Unregistering commands by command handler")

	addaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering addaccount command")

	addaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering addaccountnew command")

	removeaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering removeaccount command")

	accountlogs.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering accountlogs command")

	/*
		setpreference.UnregisterCommand(s, guildID)
		logger.Log.Info("Unregistering setpreference command")
	*/

	updateaccount.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering updateaccount command")

	accountage.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering accountage command")

	/*
		claimrewards.UnregisterCommand(s, guildID)
		logger.Log.Info("Unregistering claimavailablerewards command")
	*/

	checknow.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering checknow command")

	help.UnregisterCommand(s, guildID)
	logger.Log.Info("Unregistering help command")

	// Unregister global feedback command

	feedback.UnregisterCommand(s)
	logger.Log.Info("Unregistering global feedback command")

}
