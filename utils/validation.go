package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bwmarrin/discordgo"
)

var (
	emailRegex        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	activisionIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*[a-zA-Z0-9]$`)
	discordIDRegex    = regexp.MustCompile(`^\d{17,19}$`)
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func ValidateActivisionID(id string) error {
	if id == "" {
		return ValidationError{"activision_id", "Activision ID cannot be empty"}
	}

	if len(id) < 3 {
		return ValidationError{"activision_id", "Activision ID must be at least 3 characters long"}
	}

	if len(id) > 20 {
		return ValidationError{"activision_id", "Activision ID cannot be longer than 20 characters"}
	}

	if !activisionIDRegex.MatchString(id) {
		return ValidationError{"activision_id", "Activision ID contains invalid characters"}
	}

	return nil
}

func ValidateAccountTitle(title string) error {
	if title == "" {
		return ValidationError{"title", "Account title cannot be empty"}
	}

	title = strings.TrimSpace(title)
	if len(title) < 1 {
		return ValidationError{"title", "Account title cannot be empty"}
	}

	if len(title) > 50 {
		return ValidationError{"title", "Account title cannot be longer than 50 characters"}
	}

	return nil
}

func ValidateDiscordUserID(userID string) error {
	if userID == "" {
		return ValidationError{"user_id", "User ID cannot be empty"}
	}

	if !discordIDRegex.MatchString(userID) {
		return ValidationError{"user_id", "Invalid Discord user ID format"}
	}

	return nil
}

func ValidateNotificationType(notificationType string) (string, error) {
	if notificationType == "" {
		return "", ValidationError{"notification_type", "Notification type cannot be empty"}
	}

	notificationType = strings.ToLower(strings.TrimSpace(notificationType))

	switch notificationType {
	case "channel", "ch", "guild", "server":
		return "channel", nil
	case "dm", "direct", "private", "dms":
		return "dm", nil
	default:
		return "", ValidationError{"notification_type", "Invalid notification type. Please use 'channel' or 'dm'"}
	}
}

func ValidateCheckInterval(interval int) error {
	cfg := configuration.Get()

	if interval < 1 {
		return ValidationError{"check_interval", "Check interval must be at least 1 minute"}
	}

	if interval > 1440 {
		return ValidationError{"check_interval", "Check interval cannot be longer than 24 hours (1440 minutes)"}
	}

	if interval < cfg.Intervals.Check {
		return ValidationError{"check_interval", fmt.Sprintf("Check interval must be at least %d minutes for default users", cfg.Intervals.Check)}
	}

	return nil
}

func ValidateNotificationInterval(interval float64) error {
	if interval < 0.5 {
		return ValidationError{"notification_interval", "Notification interval must be at least 0.5 hours"}
	}

	if interval > 168 {
		return ValidationError{"notification_interval", "Notification interval cannot be longer than 7 days (168 hours)"}
	}

	return nil
}

func ValidateEmbedField(field *discordgo.MessageEmbedField) error {
	if field == nil {
		return ValidationError{"embed_field", "Embed field cannot be nil"}
	}

	if field.Name == "" {
		return ValidationError{"embed_field_name", "Embed field name cannot be empty"}
	}

	if len(field.Name) > 256 {
		return ValidationError{"embed_field_name", "Embed field name cannot be longer than 256 characters"}
	}

	if field.Value == "" {
		return ValidationError{"embed_field_value", "Embed field value cannot be empty"}
	}

	if len(field.Value) > 1024 {
		return ValidationError{"embed_field_value", "Embed field value cannot be longer than 1024 characters"}
	}

	return nil
}

func ValidateEmbed(embed *discordgo.MessageEmbed) error {
	cfg := configuration.Get()

	if embed == nil {
		return ValidationError{"embed", "Embed cannot be nil"}
	}

	if embed.Title != "" && len(embed.Title) > 256 {
		return ValidationError{"embed_title", "Embed title cannot be longer than 256 characters"}
	}

	if embed.Description != "" && len(embed.Description) > cfg.Message.EmbedDescLimit {
		return ValidationError{"embed_description", fmt.Sprintf("Embed description cannot be longer than %d characters", cfg.Message.EmbedDescLimit)}
	}

	if len(embed.Fields) > cfg.Message.MaxEmbedFields {
		return ValidationError{"embed_fields", fmt.Sprintf("Embed cannot have more than %d fields", cfg.Message.MaxEmbedFields)}
	}

	for i, field := range embed.Fields {
		if err := ValidateEmbedField(field); err != nil {
			return ValidationError{"embed_field", fmt.Sprintf("Field %d: %s", i+1, err.Error())}
		}
	}

	return nil
}

func ValidateMessage(content string) error {
	cfg := configuration.Get()

	if len(content) > cfg.Message.MaxLength {
		return ValidationError{"message_content", fmt.Sprintf("Message content cannot be longer than %d characters", cfg.Message.MaxLength)}
	}

	return nil
}

func ValidateTimeRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() {
		return ValidationError{"time_range", "Start and end times cannot be zero"}
	}

	if start.After(end) {
		return ValidationError{"time_range", "Start time cannot be after end time"}
	}

	if end.Sub(start) > 365*24*time.Hour {
		return ValidationError{"time_range", "Time range cannot be longer than 1 year"}
	}

	return nil
}

func ValidateCaptchaProvider(provider string) (string, error) {
	if provider == "" {
		return "", ValidationError{"captcha_provider", "Captcha provider cannot be empty"}
	}

	provider = strings.ToLower(strings.TrimSpace(provider))

	switch provider {
	case "capsolver", "cap", "caps":
		return "capsolver", nil
	case "ezcaptcha", "ez", "ezcap":
		return "ezcaptcha", nil
	case "2captcha", "2cap", "twocaptcha":
		return "2captcha", nil
	default:
		return "", ValidationError{"captcha_provider", "Invalid captcha provider. Please use 'capsolver', 'ezcaptcha', or '2captcha'"}
	}
}

func ValidateAPIKey(apiKey string, provider string) error {
	if apiKey == "" {
		return ValidationError{"api_key", "API key cannot be empty"}
	}

	apiKey = strings.TrimSpace(apiKey)

	switch provider {
	case "capsolver":
		if len(apiKey) < 32 || len(apiKey) > 128 {
			return ValidationError{"api_key", "Capsolver API key must be between 32 and 128 characters"}
		}
		if !regexp.MustCompile(`^CAP-[A-F0-9]+$`).MatchString(apiKey) {
			return ValidationError{"api_key", "Invalid Capsolver API key format"}
		}
	case "ezcaptcha":
		if len(apiKey) < 16 || len(apiKey) > 64 {
			return ValidationError{"api_key", "EZCaptcha API key must be between 16 and 64 characters"}
		}
		if !regexp.MustCompile(`^[a-f0-9]+$`).MatchString(apiKey) {
			return ValidationError{"api_key", "Invalid EZCaptcha API key format"}
		}
	case "2captcha":
		if len(apiKey) < 16 || len(apiKey) > 64 {
			return ValidationError{"api_key", "2Captcha API key must be between 16 and 64 characters"}
		}
		if !regexp.MustCompile(`^[a-f0-9]+$`).MatchString(apiKey) {
			return ValidationError{"api_key", "Invalid 2Captcha API key format"}
		}
	default:
		return ValidationError{"captcha_provider", "Unknown captcha provider"}
	}

	return nil
}

func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}

	if maxLength <= 3 {
		return s[:maxLength]
	}

	return s[:maxLength-3] + "..."
}

func NormalizeInput(input string) string {
	return strings.TrimSpace(strings.ReplaceAll(input, "\n", " "))
}

func IsValidChannelID(channelID string) bool {
	return discordIDRegex.MatchString(channelID)
}

func IsValidGuildID(guildID string) bool {
	return discordIDRegex.MatchString(guildID)
}
