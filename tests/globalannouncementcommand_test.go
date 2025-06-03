package globalannouncement

import (
	"testing"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestSendGlobalAnnouncement(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendGlobalAnnouncement(tt.args.s, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("SendGlobalAnnouncement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandGlobalAnnouncement(t *testing.T) {
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
			CommandGlobalAnnouncement(tt.args.s, tt.args.i)
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

func Test_sendDynamicAnnouncementToUser(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
		embed  *discordgo.MessageEmbed
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sendDynamicAnnouncementToUser(tt.args.s, tt.args.userID, tt.args.embed); (err != nil) != tt.wantErr {
				t.Errorf("sendDynamicAnnouncementToUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getChannelForAnnouncement(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		userID       string
		userSettings models.UserSettings
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
			got, err := getChannelForAnnouncement(tt.args.s, tt.args.userID, tt.args.userSettings)
			if (err != nil) != tt.wantErr {
				t.Errorf("getChannelForAnnouncement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getChannelForAnnouncement() = %v, want %v", got, tt.want)
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
