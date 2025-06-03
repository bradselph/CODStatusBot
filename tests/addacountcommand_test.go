package addaccount

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func Test_getMaxAccounts(t *testing.T) {
	type args struct {
		hasCustomKey bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMaxAccounts(tt.args.hasCustomKey); got != tt.want {
				t.Errorf("getMaxAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandAddAccount(t *testing.T) {
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
			CommandAddAccount(tt.args.s, tt.args.i)
		})
	}
}

func Test_showAddAccountModal(t *testing.T) {
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
			showAddAccountModal(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleModalSubmit(t *testing.T) {
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
			HandleModalSubmit(tt.args.s, tt.args.i)
		})
	}
}

func Test_sendFollowupMessage(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		content string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendFollowupMessage(tt.args.s, tt.args.i, tt.args.content)
		})
	}
}

func Test_sendFollowupMessageWithEmbed(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		content string
		embed   *discordgo.MessageEmbed
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendFollowupMessageWithEmbed(tt.args.s, tt.args.i, tt.args.content, tt.args.embed)
		})
	}
}

func Test_formatAccountAge(t *testing.T) {
	type args struct {
		created time.Time
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
			if got := formatAccountAge(tt.args.created); got != tt.want {
				t.Errorf("formatAccountAge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUserID(t *testing.T) {
	type args struct {
		i *discordgo.InteractionCreate
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
			if got := getUserID(tt.args.i); got != tt.want {
				t.Errorf("getUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getChannelID(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
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
			if got := getChannelID(tt.args.s, tt.args.i); got != tt.want {
				t.Errorf("getChannelID() = %v, want %v", got, tt.want)
			}
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

func Test_respondToInteraction(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondToInteraction(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}
