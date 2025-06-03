package checkcaptchabalance

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandCheckCaptchaBalance(t *testing.T) {
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
			CommandCheckCaptchaBalance(tt.args.s, tt.args.i)
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
