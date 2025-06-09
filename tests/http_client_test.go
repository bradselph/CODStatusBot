package services

import (
	"net/http"
	"reflect"
	"testing"
)

func TestInitHTTPClients(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitHTTPClients()
		})
	}
}

func TestGetDefaultHTTPClient(t *testing.T) {
	tests := []struct {
		name string
		want *http.Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDefaultHTTPClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultHTTPClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLongTimeoutHTTPClient(t *testing.T) {
	tests := []struct {
		name string
		want *http.Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLongTimeoutHTTPClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLongTimeoutHTTPClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
