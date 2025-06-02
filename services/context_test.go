package services

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestGetInstallContext(t *testing.T) {
	type args struct {
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
		want InstallContext
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInstallContext(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstallContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
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
			got, err := GetUserID(tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetResponseChannel(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
		i      *discordgo.InteractionCreate
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
			got, err := GetResponseChannel(tt.args.s, tt.args.userID, tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetResponseChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetResponseChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendMessageToUser(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		userID  string
		i       *discordgo.InteractionCreate
		message string
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
			if err := SendMessageToUser(tt.args.s, tt.args.userID, tt.args.i, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("SendMessageToUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendEmbedToUser(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		userID string
		i      *discordgo.InteractionCreate
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
			if err := SendEmbedToUser(tt.args.s, tt.args.userID, tt.args.i, tt.args.embed); (err != nil) != tt.wantErr {
				t.Errorf("SendEmbedToUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
