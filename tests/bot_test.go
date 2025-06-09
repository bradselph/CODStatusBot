package bot

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestStartBot(t *testing.T) {
	tests := []struct {
		name    string
		want    *discordgo.Session
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StartBot()
			if (err != nil) != tt.wantErr {
				t.Errorf("StartBot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartBot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getInstallationType(t *testing.T) {
	type args struct {
		i *discordgo.InteractionCreate
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
			if got := getInstallationType(tt.args.i); got != tt.want {
				t.Errorf("getInstallationType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUserIDFromInteraction(t *testing.T) {
	type args struct {
		i *discordgo.InteractionCreate
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
			if got := getUserIDFromInteraction(tt.args.i); got != tt.want {
				t.Errorf("getUserIDFromInteraction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleModalSubmit(t *testing.T) {
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
			handleModalSubmit(tt.args.s, tt.args.i)
		})
	}
}

func Test_handleMessageComponent(t *testing.T) {
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
			handleMessageComponent(tt.args.s, tt.args.i)
		})
	}
}
