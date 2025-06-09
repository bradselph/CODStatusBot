package helpapi

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandHelpApi(t *testing.T) {
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
			CommandHelpApi(tt.args.s, tt.args.i)
		})
	}
}
