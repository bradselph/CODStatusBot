package helpcookie

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandHelpCookie(t *testing.T) {
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
			CommandHelpCookie(tt.args.s, tt.args.i)
		})
	}
}
