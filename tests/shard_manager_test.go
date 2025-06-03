package services

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestNewShardManager(t *testing.T) {
	tests := []struct {
		name    string
		want    *ShardManager
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewShardManager()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewShardManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewShardManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGatewayBotInfo(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    *discordgo.GatewayBotResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGatewayBotInfo(tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGatewayBotInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGatewayBotInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardManager_StartShards(t *testing.T) {
	type args struct {
		token    string
		handlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate)
	}
	tests := []struct {
		name    string
		sm      *ShardManager
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.sm.StartShards(tt.args.token, tt.args.handlers); (err != nil) != tt.wantErr {
				t.Errorf("ShardManager.StartShards() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShardManager_Close(t *testing.T) {
	tests := []struct {
		name string
		sm   *ShardManager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sm.Close()
		})
	}
}

func TestShardManager_GetSession(t *testing.T) {
	type args struct {
		guildID string
	}
	tests := []struct {
		name string
		sm   *ShardManager
		args args
		want *discordgo.Session
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sm.GetSession(tt.args.guildID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ShardManager.GetSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_registerHandlers(t *testing.T) {
	type args struct {
		s        *discordgo.Session
		handlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate)
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerHandlers(tt.args.s, tt.args.handlers)
		})
	}
}

func Test_getShardIDForGuild(t *testing.T) {
	type args struct {
		guildID    string
		shardCount int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getShardIDForGuild(tt.args.guildID, tt.args.shardCount); got != tt.want {
				t.Errorf("getShardIDForGuild() = %v, want %v", got, tt.want)
			}
		})
	}
}
