package services

import (
	"reflect"
	"testing"
)

func TestGenerateHeaders(t *testing.T) {
	type args struct {
		ssoCookie string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateHeaders(tt.args.ssoCookie); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}
