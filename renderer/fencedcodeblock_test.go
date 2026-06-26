package renderer

import "testing"

func TestParseKeyValueInt(t *testing.T) {
	tests := []struct {
		name     string
		option   string
		key      string
		wantVal  int
		wantOk   bool
	}{
		{"valid width", "width=400", "width", 400, true},
		{"valid width large", "width=760", "width", 760, true},
		{"zero is invalid", "width=0", "width", 0, false},
		{"negative is invalid", "width=-1", "width", 0, false},
		{"not a number", "width=abc", "width", 0, false},
		{"wrong key", "scale=2", "width", 0, false},
		{"no equals", "width", "width", 0, false},
		{"empty value", "width=", "width", 0, false},
		{"float value", "width=3.5", "width", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := parseKeyValueInt(tt.option, tt.key)
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("parseKeyValueInt(%q, %q) = (%d, %v), want (%d, %v)", tt.option, tt.key, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}

func TestParseKeyValueFloat(t *testing.T) {
	tests := []struct {
		name     string
		option   string
		key      string
		wantVal  float64
		wantOk   bool
	}{
		{"valid scale int", "scale=2", "scale", 2.0, true},
		{"valid scale float", "scale=1.5", "scale", 1.5, true},
		{"valid scale large", "scale=3", "scale", 3.0, true},
		{"zero is invalid", "scale=0", "scale", 0, false},
		{"negative is invalid", "scale=-1", "scale", 0, false},
		{"not a number", "scale=abc", "scale", 0, false},
		{"wrong key", "width=2", "scale", 0, false},
		{"no equals", "scale", "scale", 0, false},
		{"empty value", "scale=", "scale", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := parseKeyValueFloat(tt.option, tt.key)
			if ok != tt.wantOk || val != tt.wantVal {
				t.Errorf("parseKeyValueFloat(%q, %q) = (%f, %v), want (%f, %v)", tt.option, tt.key, val, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}
