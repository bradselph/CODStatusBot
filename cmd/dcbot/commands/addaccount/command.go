package addaccount

import (
	"github.com/bwmarrin/discordgo"
	"sbchecker/cmd/dcbot/commands/removeaccount"
	"sbchecker/internal/database"
	"sbchecker/internal/logger"
	"sbchecker/internal/services"
	"sbchecker/models"
)

// RegisterCommand registers the "addaccount" command in the Discord session for a specific guild.
func RegisterCommand(s *discordgo.Session, guildID string) {
	// Define the command and its options.
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "addaccount",
			Description: "Add or remove an account for shadowban checking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "title",
					Description: "The title of the account",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "sso_cookie",
					Description: "The SSO cookie for the account",
					Required:    true,
				},
			},
		},
	}

	// Fetch existing commands.
	existingCommands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands")
		return
	}

	// Check if the "addaccount" command already exists.
	var existingCommand *discordgo.ApplicationCommand
	for _, command := range existingCommands {
		if command.Name == "addaccount" {
			existingCommand = command
			break
		}
	}

	newCommand := commands[0]

	// If the command exists, update it. Otherwise, create a new one.
	if existingCommand != nil {
		logger.Log.Info("Updating addaccount command")
		_, err = s.ApplicationCommandEdit(s.State.User.ID, guildID, existingCommand.ID, newCommand)
		if err != nil {
			logger.Log.WithError(err).Error("Error updating addaccount command")
			return
		}
	} else {
		logger.Log.Info("Creating addaccount command")
		_, err = s.ApplicationCommandCreate(s.State.User.ID, guildID, newCommand)
		if err != nil {
			logger.Log.WithError(err).Error("Error creating addaccount command")
			return
		}
	}
}

// UnregisterCommand removes all application commands from the Discord session for a specific guild.
func UnregisterCommand(s *discordgo.Session, guildID string) {
	// Fetch existing commands.
	commands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err != nil {
		logger.Log.WithError(err).Error("Error getting application commands")
		return
	}

	// Delete each command.
	for _, command := range commands {
		logger.Log.Infof("Deleting command %s", command.Name)
		err := s.ApplicationCommandDelete(s.State.User.ID, guildID, command.ID)
		if err != nil {
			logger.Log.WithError(err).Errorf("Error deleting command %s", command.Name)
			return
		}
	}
}

// CommandAddAccount handles the "addaccount" command when invoked.
func CommandAddAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Log.Info("Invoked addaccount command")

	// Extract the command options guild, channel, and user IDs.
	title := i.ApplicationCommandData().Options[0].StringValue()
	ssoCookie := i.ApplicationCommandData().Options[1].StringValue()
	guildID := i.GuildID
	channelID := i.ChannelID
	userID := i.Member.User.ID

	// Log the command details.
	logger.Log.WithFields(map[string]interface{}{
		"title":      title,
		"sso_cookie": ssoCookie,
		"guild_id":   guildID,
		"channel_id": channelID,
		"user_id":    userID,
	}).Info("Add account command")

	// Check if the account already exists in the database.
	var account models.Account
	result := database.DB.Where("user_id = ? AND title = ?", userID, title).First(&account)
	if result.Error == nil {
		// If the account exists, respond with an ephemeral message.
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Account already exists",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Respond with a deferred ephemeral message.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// Verify the SSO cookie and create the account in a separate goroutine.
	go func() {
		statusCode, err := services.VerifySSOCookie(ssoCookie)
		if err != nil {
			// If there's an error, respond with an ephemeral message.
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Error verifying SSO cookie",
			})
			return
		}

		// Log the verification status.
		logger.Log.WithField("status_code", statusCode).Info("SSO cookie verification status")

		// If the status code is not 200, the SSO cookie is invalid.
		if statusCode != 200 {
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Invalid SSO cookie",
			})
			return
		}

		// Create the account.
		account = models.Account{
			UserID:    userID,
			Title:     title,
			SSOCookie: ssoCookie,
			GuildID:   guildID,
			ChannelID: channelID,
		}

		// Save the account to the database.
		result := database.DB.Create(&account)
		if result.Error != nil {
			// If there's an error, log it and respond with an ephemeral message.
			logger.Log.WithError(result.Error).Error("Error creating account")
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Error creating account",
			})
			return
		}

		// Respond with an ephemeral message indicating success.
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Account added",
		})

		// Update the account choices for the "removeaccount" command.
		removeaccount.UpdateAccountChoices(s, guildID)

		// Check the account for shadowbans.
		go services.CheckSingleAccount(account, s)
	}()
}
