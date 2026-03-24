package ascii

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/kovetskiy/mark/v16/attachment"
	"github.com/stretchr/testify/assert"
)

func TestProcessASCII(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		art     []byte
		scale   float64
		want    attachment.Attachment
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:  "simple ascii art",
			title: "cat",
			art:   []byte("   /\\_/\\\n  ( o.o )\n   > ^ <\n"),
			scale: 1.0,
			want: attachment.Attachment{
				// PNG magic header
				FileBytes: []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa},
				Filename:  "cat.png",
				Name:      "cat",
				Replace:   "cat",
				Checksum:  "1410adbe197adb5ad090d82019096ae0da458009bc62cc40363f97e96447fa4f",
				ID:        "",
			},
			wantErr: assert.NoError,
		},
		{
			name:  "empty title uses checksum",
			title: "",
			art:   []byte("hello\n"),
			scale: 1.0,
			want: attachment.Attachment{
				FileBytes: []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa},
				Checksum:  "6c4e50b3665409b67590e336d7ab2f6d56e0498f3bcb1b1c89b2e34d99850ff4",
				ID:        "",
			},
			wantErr: assert.NoError,
		},
		{
			name:  "scaled rendering",
			title: "scaled",
			art:   []byte("AB\n"),
			scale: 2.0,
			want: attachment.Attachment{
				FileBytes: []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa},
				Filename:  "scaled.png",
				Name:      "scaled",
				Replace:   "scaled",
				Checksum:  "eb5af344739359f06f9e040d76351d559ff87a8f6c9573fbd77bba4c6f4acd01",
				ID:        "",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessASCII(tt.title, tt.art, tt.scale)
			if !tt.wantErr(t, err, fmt.Sprintf("ProcessASCII(%q, %q, %v)", tt.title, string(tt.art), tt.scale)) {
				return
			}

			// Validate PNG magic header.
			assert.Equal(t, tt.want.FileBytes, got.FileBytes[0:8], "PNG header")
			assert.Equal(t, tt.want.Checksum, got.Checksum, "Checksum")
			assert.Equal(t, tt.want.ID, got.ID, "ID")

			if tt.title == "" {
				// Empty title falls back to checksum for name/filename/replace.
				assert.Equal(t, got.Checksum, got.Name, "empty title -> name equals checksum")
				assert.Equal(t, got.Checksum+".png", got.Filename, "empty title -> filename uses checksum")
				assert.Equal(t, got.Checksum, got.Replace, "empty title -> replace equals checksum")
			} else {
				assert.Equal(t, tt.want.Filename, got.Filename, "Filename")
				assert.Equal(t, tt.want.Name, got.Name, "Name")
				assert.Equal(t, tt.want.Replace, got.Replace, "Replace")
			}

			// Width and height must be positive.
			gotWidth, widthErr := strconv.ParseInt(got.Width, 10, 64)
			assert.NoError(t, widthErr, "Width parse")
			assert.Greater(t, gotWidth, int64(0), "Width > 0")

			gotHeight, heightErr := strconv.ParseInt(got.Height, 10, 64)
			assert.NoError(t, heightErr, "Height parse")
			assert.Greater(t, gotHeight, int64(0), "Height > 0")
		})
	}
}
