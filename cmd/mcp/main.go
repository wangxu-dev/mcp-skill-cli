package main

import (
	"os"

	"mcp-skill-manager/internal/mcpcli"
)

func main() {
	app := mcpcli.New("mcp", os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
