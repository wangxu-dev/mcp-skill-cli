package skillcli

import (
	"fmt"
	"io"
	"strings"
)

type App struct {
	binaryName string
	out        io.Writer
	errOut     io.Writer
}

func New(binaryName string, out, errOut io.Writer) *App {
	return &App{
		binaryName: binaryName,
		out:        out,
		errOut:     errOut,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 || isHelp(args[0]) {
		a.printHelp()
		return 0
	}

	switch args[0] {
	case "install", "i":
		return a.runInstall(args[1:])
	case "list":
		return a.runList(args[1:])
	case "view":
		return a.runView(args[1:])
	case "update", "upgrade":
		return a.runUpdate(args[1:])
	case "uninstall", "remove", "rm":
		return a.runUninstall(args[1:])
	case "clean":
		return a.runClean(args[1:])
	default:
		fmt.Fprintf(a.errOut, "unknown command: %s\n", args[0])
		a.printHelp()
		return 2
	}
}

func (a *App) printHelp() {
	fmt.Fprintf(a.out, `Usage: %s <command> [options]

Commands:
  install|i <source>   Install skills from repo, local path, or local store
  list               List installed skills
  view <name>         Show installed skill metadata
  update|upgrade      Update installed skills from registry
  uninstall|remove|rm <name>   Remove an installed skill
  clean              Clear local store (~/.mcp-skill/skill and ~/.mcp-skill/mcp)

Use "%s <command> -h" for command help.
`, a.binaryName, a.binaryName)
}

func isHelp(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "-h" || value == "--help" || value == "help"
}
