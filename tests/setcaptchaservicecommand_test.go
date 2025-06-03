package setcaptchaservice

import (
	"reflect"
	"testing"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func TestCommandSetCaptchaService(t *testing.T) {
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
			CommandSetCaptchaService(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleCaptchaServiceSelection(t *testing.T) {
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
			HandleCaptchaServiceSelection(tt.args.s, tt.args.i)
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

func Test_createProviderButton(t *testing.T) {
	type args struct {
		provider string
	}
	tests := []struct {
		name string
		args args
		want discordgo.Button
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createProviderButton(tt.args.provider); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createProviderButton() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleAPIKeyRemoval(t *testing.T) {
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
			handleAPIKeyRemoval(tt.args.s, tt.args.i)
		})
	}
}

func Test_showAPIKeyModal(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		i        *discordgo.InteractionCreate
		provider string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showAPIKeyModal(tt.args.s, tt.args.i, tt.args.provider)
		})
	}
}

func Test_getAPIKeyFromModal(t *testing.T) {
	type args struct {
		data discordgo.ModalSubmitInteractionData
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
			if got := getAPIKeyFromModal(tt.args.data); got != tt.want {
				t.Errorf("getAPIKeyFromModal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateAndSaveAPIKey(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		i        *discordgo.InteractionCreate
		userID   string
		provider string
		apiKey   string
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
			if err := validateAndSaveAPIKey(tt.args.s, tt.args.i, tt.args.userID, tt.args.provider, tt.args.apiKey); (err != nil) != tt.wantErr {
				t.Errorf("validateAndSaveAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_updateAPIKeys(t *testing.T) {
	type args struct {
		settings *models.UserSettings
		provider string
		apiKey   string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateAPIKeys(tt.args.settings, tt.args.provider, tt.args.apiKey)
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

func Test_respondToInteractionWithEmbed(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
		embed   *discordgo.MessageEmbed
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondToInteractionWithEmbed(tt.args.s, tt.args.i, tt.args.message, tt.args.embed)
		})
	}
}

func Test_showFallbackSettings(t *testing.T) {
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
			showFallbackSettings(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleFallbackSettingsInteraction(t *testing.T) {
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
			HandleFallbackSettingsInteraction(tt.args.s, tt.args.i)
		})
	}
}

func Test_toggleFallbackEnabled(t *testing.T) {
	type args struct {
		s      *discordgo.Session
		i      *discordgo.InteractionCreate
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toggleFallbackEnabled(tt.args.s, tt.args.i, tt.args.userID)
		})
	}
}

func Test_setFallbackProvider(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		i        *discordgo.InteractionCreate
		userID   string
		provider string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFallbackProvider(tt.args.s, tt.args.i, tt.args.userID, tt.args.provider)
		})
	}
}
