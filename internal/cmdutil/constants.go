package cmdutil

import (
	"os"
	"strings"
)

const (
	DefaultLimit         = 25
	DefaultChildrenLimit = 50

	TitleTruncateShort  = 40
	TitleTruncateNormal = 50
	TitleTruncateLong   = 60

	FilePermRW      os.FileMode = 0644
	FilePermRWX     os.FileMode = 0755
	DirPermStandard os.FileMode = 0755
)

var ValidImageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".webp": true,
}

func IsValidImageExt(ext string) bool {
	return ValidImageExts[strings.ToLower(ext)]
}
