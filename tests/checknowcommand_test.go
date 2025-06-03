package checknow

import (
	"testing"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestCommandCheckNow(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CommandCheckNow(tt.args.s, tt.args.i)
		})
	}
}

func Test_showAccountButtons(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		i            *discordgo.InteractionCreate
		accounts     []models.Account
		userSettings models.UserSettings
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showAccountButtons(tt.args.s, tt.args.i, tt.args.accounts, tt.args.userSettings)
		})
	}
}

func TestHandleAccountSelection(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleAccountSelection(tt.args.s, tt.args.i)
		})
	}
}

func Test_formatDuration(t *testing.T) {
	type args struct {
		d time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.args.d); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_respondToInteractionWithEmbed(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		i         *discordgo.InteractionCreate
		content   string
		embed     *discordgo.MessageEmbed
		ephemeral bool
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondToInteractionWithEmbed(tt.args.s, tt.args.i, tt.args.content, tt.args.embed, tt.args.ephemeral)
		})
	}
}

func Test_checkAccounts(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		i            *discordgo.InteractionCreate
		accounts     []models.Account
		userSettings models.UserSettings
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkAccounts(tt.args.s, tt.args.i, tt.args.accounts, tt.args.userSettings)
		})
	}
}

func Test_checkRateLimit(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkRateLimit(tt.args.userID); got != tt.want {
				t.Errorf("checkRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUserID(t *testing.T) {
	type args struct {
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getUserID(tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_respondToInteraction(t *testing.T) {
	type args struct {
		s              *discordgo.Session
		i              *discordgo.InteractionCreate
		message        string
		forceEphemeral bool
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondToInteraction(tt.args.s, tt.args.i, tt.args.message, tt.args.forceEphemeral)
		})
	}
}
