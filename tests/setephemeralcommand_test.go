package setephemeral

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandSetEphemeral(t *testing.T) {
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
			CommandSetEphemeral(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleEphemeralSelection(t *testing.T) {
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
			HandleEphemeralSelection(tt.args.s, tt.args.i)
		})
	}
}

func Test_respondWithError(t *testing.T) {
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
			respondWithError(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}
