package poller

import (
	"testing"
)

func TestHexToNNumber(t *testing.T) {
	tests := []struct {
		hex      string
		expected string
	}{
		{"a00001", "N1"},
		{"abcdef", "N86QU"},
		{"abc0e4", "N8566Z"},
		{"adf7c7", "N99999"},
		{"39de4a", ""}, // French hex
		{"781234", ""}, // Chinese hex
		{"invalid", ""},
	}

	for _, tt := range tests {
		actual := HexToNNumber(tt.hex)
		if actual != tt.expected {
			t.Errorf("HexToNNumber(%s) = %s, expected %s", tt.hex, actual, tt.expected)
		}
	}
}

func TestHexToCountry(t *testing.T) {
	tests := []struct {
		hex      string
		expected string
	}{
		{"a00001", "United States"},
		{"39de4a", "France"},
		{"781234", "China"},
		{"840001", "Japan"},
		{"3c0001", "Germany"},
		{"400001", "United Kingdom"},
		{"c00001", "Canada"},
	}

	for _, tt := range tests {
		actual := HexToCountry(tt.hex)
		if actual != tt.expected {
			t.Errorf("HexToCountry(%s) = %s, expected %s", tt.hex, actual, tt.expected)
		}
	}
}
