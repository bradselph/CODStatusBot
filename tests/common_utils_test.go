package utils

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func TestRespondToInteraction(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		i         *discordgo.InteractionCreate
		message   string
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
			RespondToInteraction(tt.args.s, tt.args.i, tt.args.message, tt.args.ephemeral)
		})
	}
}

func TestRespondWithEmbed(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		i         *discordgo.InteractionCreate
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
			RespondWithEmbed(tt.args.s, tt.args.i, tt.args.embed, tt.args.ephemeral)
		})
	}
}

func TestUpdateMessageWithEmbed(t *testing.T) {
	type args struct {
		s     *discordgo.Session
		i     *discordgo.InteractionCreate
		embed *discordgo.MessageEmbed
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateMessageWithEmbed(tt.args.s, tt.args.i, tt.args.embed)
		})
	}
}

func TestSendFollowupMessage(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		i         *discordgo.InteractionCreate
		message   string
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
			SendFollowupMessage(tt.args.s, tt.args.i, tt.args.message, tt.args.ephemeral)
		})
	}
}

func TestWithTransaction(t *testing.T) {
	type args struct {
		db *gorm.DB
		fn func(*gorm.DB) error
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
			if err := WithTransaction(tt.args.db, tt.args.fn); (err != nil) != tt.wantErr {
				t.Errorf("WithTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
