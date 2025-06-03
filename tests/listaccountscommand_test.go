package listaccounts

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandListAccounts(t *testing.T) {
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
			CommandListAccounts(tt.args.s, tt.args.i)
		})
	}
}

func Test_sendFollowup(t *testing.T) {
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
			sendFollowup(tt.args.s, tt.args.i, tt.args.content)
		})
	}
}

func Test_getDisabledEmoji(t *testing.T) {
	type args struct {
		isDisabled bool
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
			if got := getDisabledEmoji(tt.args.isDisabled); got != tt.want {
				t.Errorf("getDisabledEmoji() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBalanceInfo(t *testing.T) {
	type args struct {
		userID string
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
			if got := getBalanceInfo(tt.args.userID); got != tt.want {
				t.Errorf("getBalanceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
