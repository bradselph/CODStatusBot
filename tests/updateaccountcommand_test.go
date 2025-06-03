package updateaccount

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestCommandUpdateAccount(t *testing.T) {
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
			CommandUpdateAccount(tt.args.s, tt.args.i)
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

func Test_processAccountUpdate(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		i            *discordgo.InteractionCreate
		accountID    int
		newSSOCookie string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processAccountUpdate(tt.args.s, tt.args.i, tt.args.accountID, tt.args.newSSOCookie)
		})
	}
}

func Test_createSuccessEmbed(t *testing.T) {
	type args struct {
		account             *models.Account
		wasDisabled         bool
		vipStatusChange     string
		expirationTimestamp int64
		isVIP               bool
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
			if got := createSuccessEmbed(tt.args.account, tt.args.wasDisabled, tt.args.vipStatusChange, tt.args.expirationTimestamp, tt.args.isVIP); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createSuccessEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVIPStatusText(t *testing.T) {
	type args struct {
		isVIP bool
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
			if got := getVIPStatusText(tt.args.isVIP); got != tt.want {
				t.Errorf("getVIPStatusText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendFollowupMessage(t *testing.T) {
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
			sendFollowupMessage(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}

func Test_sendFollowupMessageWithEmbed(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
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
			sendFollowupMessageWithEmbed(tt.args.s, tt.args.i, tt.args.message, tt.args.embed)
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

func Test_respondToInteractionWithMessage(t *testing.T) {
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
			respondToInteractionWithMessage(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}
