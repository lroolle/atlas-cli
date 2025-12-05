package main

import (
	"github.com/lroolle/atlas-cli/cmd"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cmd.Execute()
}
