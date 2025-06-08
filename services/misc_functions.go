package services

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func GetUserID(i *discordgo.InteractionCreate) (string, error) {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID, nil
	}
	if i.User != nil {
		return i.User.ID, nil
	}
	return "", fmt.Errorf("unable to determine user ID")
}

func formatVIPStatus(isVIP bool) string {
	if isVIP {
		return "Yes"
	}
	return "No"
}

func formatCheckStatus(isDisabled bool) string {
	if isDisabled {
		return "DISABLED"
	}
	return "ENABLED"
}
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
