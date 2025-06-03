package command

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestRegisterCommands(t *testing.T) {
	type args struct {
		s *discordgo.Session
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
			if err := RegisterCommands(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("RegisterCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleCommand(t *testing.T) {
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
			HandleCommand(tt.args.s, tt.args.i)
		})
	}
}

func TestBoolPtr(t *testing.T) {
	type args struct {
		b bool
	}
	tests := []struct {
		name string
		args args
		want *bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BoolPtr(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BoolPtr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64Ptr(t *testing.T) {
	type args struct {
		i int64
	}
	tests := []struct {
		name string
		args args
		want *int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64Ptr(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Int64Ptr() = %v, want %v", got, tt.want)
			}
		})
	}
}
