package services

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestTrackUserInteraction(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
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
			if err := TrackUserInteraction(tt.args.s, tt.args.i); (err != nil) != tt.wantErr {
				t.Errorf("TrackUserInteraction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotifyNewInstallation(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		context string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NotifyNewInstallation(tt.args.s, tt.args.context)
		})
	}
}

func TestGetInstallationStats(t *testing.T) {
	tests := []struct {
		name            string
		wantServerCount int64
		wantDirectCount int64
		wantErr         bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServerCount, gotDirectCount, err := GetInstallationStats()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstallationStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotServerCount != tt.wantServerCount {
				t.Errorf("GetInstallationStats() gotServerCount = %v, want %v", gotServerCount, tt.wantServerCount)
			}
			if gotDirectCount != tt.wantDirectCount {
				t.Errorf("GetInstallationStats() gotDirectCount = %v, want %v", gotDirectCount, tt.wantDirectCount)
			}
		})
	}
}

func TestLogInstallationStats(t *testing.T) {
	type args struct {
		s *discordgo.Session
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LogInstallationStats(tt.args.s)
		})
	}
}
