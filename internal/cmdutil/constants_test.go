package cmdutil

import "testing"

func TestIsValidImageExt(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".PNG", true},
		{".jpg", true},
		{".JPG", true},
		{".jpeg", true},
		{".JPEG", true},
		{".gif", true},
		{".GIF", true},
		{".svg", true},
		{".webp", true},
		{".txt", false},
		{".pdf", false},
		{".doc", false},
		{"", false},
		{"png", false},
		{".pnga", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := IsValidImageExt(tt.ext); got != tt.want {
				t.Errorf("IsValidImageExt(%q) = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestFilePermissions(t *testing.T) {
	if FilePermRW != 0644 {
		t.Errorf("FilePermRW = %o, want 0644", FilePermRW)
	}
	if FilePermRWX != 0755 {
		t.Errorf("FilePermRWX = %o, want 0755", FilePermRWX)
	}
	if DirPermStandard != 0755 {
		t.Errorf("DirPermStandard = %o, want 0755", DirPermStandard)
	}
}
