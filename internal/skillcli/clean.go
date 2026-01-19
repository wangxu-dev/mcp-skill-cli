package skillcli

import (
	"flag"
	"fmt"

	"mcp-skill-manager/internal/skill"
)

func (a *App) runClean(args []string) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	helpShort := fs.Bool("h", false, "show help")
	helpLong := fs.Bool("help", false, "show help")

	flags, _ := splitArgs(args)
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if *helpShort || *helpLong {
		a.printCleanHelp()
		return 0
	}

	skillRoot, mcpRoot, err := skill.LocalStorePaths()
	if err != nil {
		fmt.Fprintf(a.errOut, "clean failed: %v\n", err)
		return 1
	}

	skillCount, err := countEntries(skillRoot)
	if err != nil {
		fmt.Fprintf(a.errOut, "clean failed: %v\n", err)
		return 1
	}
	mcpCount, err := countEntries(mcpRoot)
	if err != nil {
		fmt.Fprintf(a.errOut, "clean failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(a.out, "Local store will be cleared:\n- %s (%d item(s))\n- %s (%d item(s))\n", skillRoot, skillCount, mcpRoot, mcpCount)
	if !confirmPrompt(a.out, "Type 'yes' to continue: ") {
		fmt.Fprintln(a.out, "canceled")
		return 0
	}

	if err := skill.CleanLocalStore(); err != nil {
		fmt.Fprintf(a.errOut, "clean failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(a.out, "clean complete")
	return 0
}

func (a *App) printCleanHelp() {
	fmt.Fprintf(a.out, `Usage: %s clean

Clears:
  ~/.mcp-skill/skill
  ~/.mcp-skill/mcp
`, a.binaryName)
}
