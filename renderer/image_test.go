package renderer

import "testing"

func TestCalculateAlign(t *testing.T) {
	tests := []struct {
		name            string
		configuredAlign string
		width           string
		expectedAlign   string
	}{
		{"No alignment configured", "", "1000", ""},
		{"Center alignment small", "center", "500", "center"},
		{"Left alignment small", "left", "500", "left"},
		{"Right alignment small", "right", "500", "right"},
		{"Left stays left at 760px", "left", "760", "left"},
		{"Left stays left above 760px", "left", "1000", "left"},
		{"Right stays right at 1800px", "right", "1800", "right"},
		{"No width provided", "left", "", "left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAlign(tt.configuredAlign, tt.width)
			if result != tt.expectedAlign {
				t.Errorf("calculateAlign(%q, %q) = %q, want %q", tt.configuredAlign, tt.width, result, tt.expectedAlign)
			}
		})
	}
}

func TestCalculateLayout(t *testing.T) {
	tests := []struct {
		name           string
		align          string
		width          string
		expectedLayout string
	}{
		// Small images use alignment-based layout
		{"Left alignment small", "left", "500", "align-start"},
		{"Center alignment small", "center", "500", "center"},
		{"Right alignment small", "right", "500", "align-end"},
		{"No alignment small", "", "500", ""},

		// Large images also use alignment-based layout (no more wide/full-width)
		{"Left at 760px", "left", "760", "align-start"},
		{"Center at 1000px", "center", "1000", "center"},
		{"Right at 1799px", "right", "1799", "align-end"},
		{"Center at 1800px", "center", "1800", "center"},
		{"Center at 2000px", "center", "2000", "center"},

		// Edge cases
		{"No width", "center", "", "center"},
		{"Invalid width", "center", "abc", "center"},
		{"Empty alignment and width", "", "", ""},
		{"No alignment configured", "", "1000", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateLayout(tt.align, tt.width)
			if result != tt.expectedLayout {
				t.Errorf("calculateLayout(%q, %q) = %q, want %q", tt.align, tt.width, result, tt.expectedLayout)
			}
		})
	}
}

func TestCalculateDisplayWidth(t *testing.T) {
	tests := []struct {
		name          string
		originalWidth string
		layout        string
		expectedWidth string
	}{
		{"Large width capped to 760", "2000", "center", "760"},
		{"Width at 761 capped to 760", "761", "center", "760"},
		{"Width at 760 stays", "760", "center", "760"},
		{"Small width stays", "500", "align-start", "500"},
		{"Empty original", "", "center", ""},
		{"Empty layout", "1000", "", "760"},
		{"Invalid width passed through", "abc", "center", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDisplayWidth(tt.originalWidth, tt.layout)
			if result != tt.expectedWidth {
				t.Errorf("calculateDisplayWidth(%q, %q) = %q, want %q", tt.originalWidth, tt.layout, result, tt.expectedWidth)
			}
		})
	}
}
