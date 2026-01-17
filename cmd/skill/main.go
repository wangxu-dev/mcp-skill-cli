package main

import (
	"os"

	"mcp-skill-manager/internal/cli"
)

func main() {
	app := cli.New("skill", os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
