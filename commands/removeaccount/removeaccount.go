package removeaccount

import (
	"codstatusbot/database"
	"codstatusbot/internal"
	"codstatusbot/internal/logger"
	"codstatusbot/models"

	"github.com/bwmarrin/discordgo"
)

// choices holds the choices for the "removeaccount" command.
var choices []*discordgo.ApplicationCommandOptionChoice

// RegisterCommand registers the "removeaccount" command in the Discord session for a specific guild.
func RegisterCommand(s *discordgo.Session, guildID string) {
	// Fetch all choices for the guild.
	choices = internal.GetAllChoices(guildID)

	// Define the command and its options.
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "removeaccount",
			Description: "Remove an account from shadowban checking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionType(discordgo.InteractionApplicationCommandAutocomplete),
					Name:        "account",
					Description: "The title of the account",
					Required:    true,
					Choices:     choices,
				},
			},
		},
	}

	// Fetch existing commands.
	existingCommands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands from guild")
		return
	}

	// Check if the "removeaccount" command already exists.
	var existingCommand *discordgo.ApplicationCommand
	for _, command := range existingCommands {
		if command.Name == "removeaccount" {
			existingCommand = command
			break
		}
	}

	newCommand := commands[0]

	// If the command exists, update it. Otherwise, create a new one.
	if existingCommand != nil {
		logger.Log.Info("Updating the removeaccount command")
		_, err = s.ApplicationCommandEdit(s.State.User.ID, guildID, existingCommand.ID, newCommand)
		if err != nil {
			logger.Log.WithError(err).Error("Error updating the removeaccount command")
			return
		}
	} else {
		logger.Log.Info("Adding the removeaccount command")
		_, err = s.ApplicationCommandCreate(s.State.User.ID, guildID, newCommand)
		if err != nil {
			logger.Log.WithError(err).Error("Error adding the removeaccount command")
			return
		}
	}
}

// UnregisterCommand removes all application commands from the Discord session for a specific guild.
func UnregisterCommand(s *discordgo.Session, guildID string) {
	// Fetch existing commands.
	commands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands for unregistering removeaccount command")
		return
	}

	// Delete each command.
	for _, command := range commands {
		logger.Log.Infof("Deleting command %s", command.Name)
		err := s.ApplicationCommandDelete(s.State.User.ID, guildID, command.ID)
		if err != nil {
			logger.Log.WithError(err).Errorf("Error unregistering the command %s", command.Name)
			continue
		}
	}
}

// CommandRemoveAccount handles the "removeaccount" command when invoked.
func CommandRemoveAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Log.Info("Starting remove account command")
	// Extract the user ID, guild ID, and account ID from the interaction.
	userID := i.Member.User.ID
	guildID := i.GuildID
	accountId := i.ApplicationCommandData().Options[0].IntValue()
	logger.Log.Infof("User ID: %s, Guild ID: %s  Account ID: %d", userID, guildID, accountId)

	// Begin a database transaction.
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch the account from the database.
	var account models.Account
	result := tx.Where("user_id = ? AND id = ? AND guild_id = ?", userID, accountId, guildID).First(&account)
	if result.Error != nil {
		// If the account does not exist, respond with an ephemeral message.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Account does not exist",
				Flags:   64, // Set ephemeral flag
			},
		})
		tx.Rollback()
		return
	}
	// Disable foreign key constraints.
	err := tx.Exec("SET FOREIGN_KEY_CHECKS=0;").Error
	if err != nil {
		logger.Log.WithError(err).Error("Error disabling foreign key constraints")
		tx.Rollback()
		return
	}
	defer tx.Exec("SET FOREIGN_KEY_CHECKS=1;") // Re-enable foreign key constraints

	// Delete associated bans.
	if err := tx.Unscoped().Where("account_id = ?", account.ID).Delete(&models.Ban{}).Error; err != nil {
		logger.Log.WithError(err).Error("Error deleting associated bans for account", account.ID)
		tx.Rollback()
		return
	}

	// Delete the account.
	if err := tx.Unscoped().Where("id = ?", account.ID).Delete(&models.Account{}).Error; err != nil {
		logger.Log.WithError(err).Error("Error deleting account from database", account.ID)
		tx.Rollback()
		return
	}

	// Respond with an ephemeral message indicating success.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Account removed",
			Flags:   64, // Set ephemeral flag
		},
	})

	// Update the choices for the "removeaccount" command.
	UpdateAccountChoices(s, guildID)
	// Commit the transaction.
	tx.Commit()
	logger.Log.Info("Account removed successfully")
}

// UpdateAccountChoices updates the choices for the "removeaccount" command and other related commands.
func UpdateAccountChoices(s *discordgo.Session, guildID string) {
	logger.Log.Info("Updating account choices")

	// Fetch all choices for the guild.
	choices = internal.GetAllChoices(guildID)

	// Fetch existing commands.
	commands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application command choices")
		return
	}

	// Define the configurations for the commands that need to be updated.
	commandConfigs := map[string]*discordgo.ApplicationCommand{
		"removeaccount": {
			Name:        "removeaccount",
			Description: "Remove an account from automated shadowban checking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionType(discordgo.InteractionApplicationCommandAutocomplete),
					Name:        "account",
					Description: "The title of the account",
					Required:    true,
					Choices:     choices,
				},
			},
		},
		"accountlogs": {
			Name:        "accountlogs",
			Description: "View the logs for an account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionType(discordgo.InteractionApplicationCommandAutocomplete),
					Name:        "account",
					Description: "The title of the account",
					Required:    true,
					Choices:     choices,
				},
			},
		},
		"updateaccount": {
			Name:        "updateaccount",
			Description: "Update the SSO cookie for an account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionType(discordgo.InteractionApplicationCommandAutocomplete),
					Name:        "account",
					Description: "The title of the account",
					Required:    true,
					Choices:     choices,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "sso_cookie",
					Description: "The new SSO cookie for the account",
					Required:    true,
				},
			},
		},
		"accountage": {
			Name:        "accountage",
			Description: "Check the age of an account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionType(discordgo.InteractionApplicationCommandAutocomplete),
					Name:        "account",
					Description: "The title of the account",
					Required:    true,
					Choices:     choices,
				},
			},
		},
	}

	// Update each command.
	for _, command := range commands {
		if config, ok := commandConfigs[command.Name]; ok {
			logger.Log.Infof("Updating command %s", command.Name)
			_, err := s.ApplicationCommandEdit(s.State.User.ID, guildID, command.ID, config)
			if err != nil {
				logger.Log.WithError(err).Errorf("Error updating command %s", command.Name)
				return
			}
		}
	}
}