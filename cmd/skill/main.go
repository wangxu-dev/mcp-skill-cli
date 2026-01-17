package main

import (
	"os"

	"mcp-skill-manager/internal/skillcli"
)

func main() {
	app := skillcli.New("skill", os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
