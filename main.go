package main

import (
	"os"

	"github.com/rtxnik/lazyray/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	os.Exit(cmd.Execute())
}
