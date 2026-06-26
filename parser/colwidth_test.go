package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWidths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"percentages", "30%, 70%", []string{"30%", "70%"}},
		{"pixels explicit", "100px,200px,300px", []string{"100px", "200px", "300px"}},
		{"pixels with spaces", " 25% , 50% , 25% ", []string{"25%", "50%", "25%"}},
		{"auto keyword", "auto, 200px", []string{"auto", "200px"}},
		{"bare numbers become px", "100, 200, 300", []string{"100px", "200px", "300px"}},
		{"bare decimal becomes px", "10.5", []string{"10.5px"}},
		{"mixed units", "30%, 200px, auto", []string{"30%", "200px", "auto"}},
		{"em and rem", "10em, 20rem", []string{"10em", "20rem"}},
		{"empty string", "", nil},
		{"only commas", " , , ", nil},
		{"invalid values filtered", "30%, bogus!!, 70%", []string{"30%", "70%"}},
		{"injection attempt filtered", `30%; background:red`, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseWidths(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeWidth(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"100", "100px"},
		{"10.5", "10.5px"},
		{"100px", "100px"},
		{"30%", "30%"},
		{"auto", "auto"},
		{"10em", "10em"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeWidth(tc.input))
		})
	}
}

func TestIsValidCSSWidth(t *testing.T) {
	valid := []string{"100px", "30%", "10.5em", "20rem", "auto", "100", "5ch", "50vw", "50vh"}
	invalid := []string{"100px; background:red", "expression(alert())", "url(x)", "10 px", "abc"}

	for _, s := range valid {
		assert.True(t, isValidCSSWidth(s), "expected valid: %s", s)
	}
	for _, s := range invalid {
		assert.False(t, isValidCSSWidth(s), "expected invalid: %s", s)
	}
}
