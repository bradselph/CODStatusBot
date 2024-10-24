package feedback

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/logger"

	"github.com/bwmarrin/discordgo"
)

var tempFeedbackStore = struct {
	sync.RWMutex
	m map[string]feedbackEntry
}{m: make(map[string]feedbackEntry)}

type feedbackEntry struct {
	message   string
	timestamp time.Time
}

const feedbackTimeout = 5 * time.Minute

func CommandFeedback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	feedbackMessage := i.ApplicationCommandData().Options[0].StringValue()
	developerID := os.Getenv("DEVELOPER_ID")
	if developerID == "" {
		logger.Log.Error("DEVELOPER_ID not set in environment variables")
		sendResponse(s, i, "Configuration error. Please try again later.", true)
		return
	}

	userID, err := getUserID(i)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get user ID")
		sendResponse(s, i, "An error occurred while processing your request.", true)
		return
	}

	// Store the feedback message temporarily
	tempFeedbackStore.Lock()
	tempFeedbackStore.m[userID] = feedbackEntry{
		message:   feedbackMessage,
		timestamp: time.Now(),
	}
	tempFeedbackStore.Unlock()

	logger.Log.WithField("userID", userID).Info("Stored feedback message")

	// Create a message with buttons for anonymity choice.
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Do you want to send this feedback anonymously?",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Send Anonymously",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("feedback_anonymous_%s", userID),
						},
						discordgo.Button{
							Label:    "Send with ID",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("feedback_with_id_%s", userID),
						},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send anonymity choice message")
		sendResponse(s, i, "There was an error processing your feedback. Please try again later.", true)
		return
	}
}

func HandleFeedbackChoice(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.SplitN(customID, "_", 3)
	if len(parts) != 3 {
		logger.Log.Error("Invalid custom ID format for feedback choice")
		sendResponse(s, i, "An error occurred while processing your request.", true)
		return
	}

	isAnonymous := parts[1] == "anonymous"
	userID := parts[2]

	userID = strings.TrimPrefix(userID, "id_")

	interactionUserID, err := getUserID(i)
	if err != nil || interactionUserID != userID {
		logger.Log.WithField("buttonUserID", userID).WithField("interactionUserID", interactionUserID).Error("User ID mismatch")
		sendResponse(s, i, "An error occurred while processing your request.", true)
		return
	}

	tempFeedbackStore.RLock()
	entry, ok := tempFeedbackStore.m[userID]
	tempFeedbackStore.RUnlock()

	if !ok || time.Since(entry.timestamp) > feedbackTimeout {
		logger.Log.WithField("userID", userID).Error("Feedback message not found or expired")
		sendResponse(s, i, "Your feedback session has expired. Please submit your feedback again.", true)
		return
	}

	// Remove the feedback message from temporary storage
	tempFeedbackStore.Lock()
	delete(tempFeedbackStore.m, userID)
	tempFeedbackStore.Unlock()

	var feedbackToSend string
	if isAnonymous {
		feedbackToSend = fmt.Sprintf("Anonymous Feedback:\n\n%s", entry.message)
	} else {
		feedbackToSend = fmt.Sprintf("Feedback from User ID %s:\n\n%s", userID, entry.message)
	}

	if err := sendFeedbackToDeveloper(s, feedbackToSend); err != nil {
		logger.Log.WithError(err).Error("Failed to send feedback to developer")
		sendResponse(s, i, "There was an error sending your feedback. Please try again later.", true)
		return
	}

	// Respond to user
	sendResponse(s, i, "Your feedback has been sent to the developer. Thank you for your input!", true)
}

func sendFeedbackToDeveloper(s *discordgo.Session, feedback string) error {
	developerID := os.Getenv("DEVELOPER_ID")
	channel, err := s.UserChannelCreate(developerID)
	if err != nil {
		return fmt.Errorf("failed to create DM channel with developer: %w", err)
	}

	_, err = s.ChannelMessageSend(channel.ID, feedback)
	if err != nil {
		return fmt.Errorf("failed to send feedback to developer: %w", err)
	}

	return nil
}

func sendResponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
	if err != nil {
		logger.Log.WithError(err).Error("Failed to send interaction response")
	}
}

func getUserID(i *discordgo.InteractionCreate) (string, error) {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID, nil
	}
	if i.User != nil {
		return i.User.ID, nil
	}
	return "", fmt.Errorf("unable to determine user ID")
}

func cleanupExpiredFeedback() {
	tempFeedbackStore.Lock()
	defer tempFeedbackStore.Unlock()

	now := time.Now()
	for userID, entry := range tempFeedbackStore.m {
		if now.Sub(entry.timestamp) > feedbackTimeout {
			delete(tempFeedbackStore.m, userID)
			logger.Log.WithField("userID", userID).Info("Removed expired feedback entry")
		}
	}
}
