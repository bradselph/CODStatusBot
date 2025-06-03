package main

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func Test_loadEnv(t *testing.T) {
	type args struct {
		filename string
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
			if err := loadEnv(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("loadEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}

func Test_run(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := run(); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_startPeriodicTasks(t *testing.T) {
	type args struct {
		ctx context.Context
		s   *discordgo.Session
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startPeriodicTasks(tt.args.ctx, tt.args.s)
		})
	}
}
