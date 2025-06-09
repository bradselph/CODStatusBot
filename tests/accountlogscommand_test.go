package accountlogs

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestCommandAccountLogs(t *testing.T) {
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
			CommandAccountLogs(tt.args.s, tt.args.i)
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

func Test_handleAllAccountLogs(t *testing.T) {
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
			handleAllAccountLogs(tt.args.s, tt.args.i)
		})
	}
}

func Test_createAccountLogEmbed(t *testing.T) {
	type args struct {
		account models.Account
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createAccountLogEmbed(tt.args.account); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createAccountLogEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatTimeField(t *testing.T) {
	type args struct {
		timestamp int64
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
			if got := formatTimeField(tt.args.timestamp); got != tt.want {
				t.Errorf("formatTimeField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_respondToInteraction(t *testing.T) {
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
			respondToInteraction(tt.args.s, tt.args.i, tt.args.content)
		})
	}
}
