package utils

import "testing"

func TestSanitizeInput(t *testing.T) {
	type args struct {
		input string
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
			if got := SanitizeInput(tt.args.input); got != tt.want {
				t.Errorf("SanitizeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeAnnouncement(t *testing.T) {
	type args struct {
		input string
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
			if got := SanitizeAnnouncement(tt.args.input); got != tt.want {
				t.Errorf("SanitizeAnnouncement() = %v, want %v", got, tt.want)
			}
		})
	}
}
