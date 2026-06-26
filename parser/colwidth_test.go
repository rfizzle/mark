package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWidths(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"30%, 70%", []string{"30%", "70%"}},
		{"100px,200px,300px", []string{"100px", "200px", "300px"}},
		{" 25% , 50% , 25% ", []string{"25%", "50%", "25%"}},
		{"auto, 200px", []string{"auto", "200px"}},
		{"", nil},
		{" , , ", nil},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseWidths(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
