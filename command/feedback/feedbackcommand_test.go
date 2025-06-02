package feedback

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandFeedback(t *testing.T) {
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
			CommandFeedback(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleFeedbackChoice(t *testing.T) {
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
			HandleFeedbackChoice(tt.args.s, tt.args.i)
		})
	}
}

func Test_sendFeedbackToDeveloper(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		feedback string
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
			if err := sendFeedbackToDeveloper(tt.args.s, tt.args.feedback); (err != nil) != tt.wantErr {
				t.Errorf("sendFeedbackToDeveloper() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sendResponse(t *testing.T) {
	type args struct {
		s         *discordgo.Session
		i         *discordgo.InteractionCreate
		content   string
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
			sendResponse(tt.args.s, tt.args.i, tt.args.content, tt.args.ephemeral)
		})
	}
}

func Test_getUserID(t *testing.T) {
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
			got, err := getUserID(tt.args.i)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}
