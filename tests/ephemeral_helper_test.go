package services

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestGetUserEphemeralPreference(t *testing.T) {
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
			if got := GetUserEphemeralPreference(tt.args.userID); got != tt.want {
				t.Errorf("GetUserEphemeralPreference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInteractionFlags(t *testing.T) {
	type args struct {
		i              *discordgo.InteractionCreate
		forceEphemeral bool
	}
	tests := []struct {
		name string
		args args
		want discordgo.MessageFlags
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInteractionFlags(tt.args.i, tt.args.forceEphemeral); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInteractionFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRespondWithPreference(t *testing.T) {
	type args struct {
		s              *discordgo.Session
		i              *discordgo.InteractionCreate
		content        string
		embeds         []*discordgo.MessageEmbed
		forceEphemeral bool
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
			if err := RespondWithPreference(tt.args.s, tt.args.i, tt.args.content, tt.args.embeds, tt.args.forceEphemeral); (err != nil) != tt.wantErr {
				t.Errorf("RespondWithPreference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRespondWithPreferenceAndComponents(t *testing.T) {
	type args struct {
		s              *discordgo.Session
		i              *discordgo.InteractionCreate
		content        string
		embeds         []*discordgo.MessageEmbed
		components     []discordgo.MessageComponent
		forceEphemeral bool
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
			if err := RespondWithPreferenceAndComponents(tt.args.s, tt.args.i, tt.args.content, tt.args.embeds, tt.args.components, tt.args.forceEphemeral); (err != nil) != tt.wantErr {
				t.Errorf("RespondWithPreferenceAndComponents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeferWithPreference(t *testing.T) {
	type args struct {
		s              *discordgo.Session
		i              *discordgo.InteractionCreate
		forceEphemeral bool
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
			if err := DeferWithPreference(tt.args.s, tt.args.i, tt.args.forceEphemeral); (err != nil) != tt.wantErr {
				t.Errorf("DeferWithPreference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFollowupWithPreference(t *testing.T) {
	type args struct {
		s              *discordgo.Session
		i              *discordgo.InteractionCreate
		content        string
		embeds         []*discordgo.MessageEmbed
		components     []discordgo.MessageComponent
		forceEphemeral bool
	}
	tests := []struct {
		name    string
		args    args
		want    *discordgo.Message
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FollowupWithPreference(tt.args.s, tt.args.i, tt.args.content, tt.args.embeds, tt.args.components, tt.args.forceEphemeral)
			if (err != nil) != tt.wantErr {
				t.Errorf("FollowupWithPreference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FollowupWithPreference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateMessageWithPreference(t *testing.T) {
	type args struct {
		s          *discordgo.Session
		i          *discordgo.InteractionCreate
		content    string
		embeds     []*discordgo.MessageEmbed
		components []discordgo.MessageComponent
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
			if err := UpdateMessageWithPreference(tt.args.s, tt.args.i, tt.args.content, tt.args.embeds, tt.args.components); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMessageWithPreference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
