package skillcli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/skill"
)

func (a *App) runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	scope := fs.String("scope", "", "deprecated: use --global or --local")
	globalShort := fs.Bool("g", false, "remove from user/global scope")
	globalLong := fs.Bool("global", false, "remove from user/global scope")
	localShort := fs.Bool("l", false, "remove from project/local scope")
	localLong := fs.Bool("local", false, "remove from project/local scope")
	projectLong := fs.Bool("project", false, "remove from project/local scope")
	forceShort := fs.Bool("f", false, "ignore missing skills")
	forceLong := fs.Bool("force", false, "ignore missing skills")
	allShort := fs.Bool("a", false, "remove for all clients")
	allLong := fs.Bool("all", false, "remove for all clients")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode,cursor,amp,kilocode,roo,goose,antigravity,copilot,clawdbot,droid,windsurf")
	clientShort := fs.String("c", "", "alias for --client")
	toolFlag := fs.String("tool", "", "deprecated: use --client")
	helpShort := fs.Bool("h", false, "show help")
	helpLong := fs.Bool("help", false, "show help")

	flags, positionals := splitArgs(args)
	parseArgs := append(flags, positionals...)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}
	if *helpShort || *helpLong {
		a.printUninstallHelp()
		return 0
	}

	allRequested := *allShort || *allLong
	clientValue, err := resolveClientValue(*clientFlag, *clientShort, *toolFlag, allRequested)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	tools, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}

	normalizedScope, err := resolveScope(*scope, *globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid scope: %v\n", err)
		return 2
	}

	cwd, _ := os.Getwd()
	var records []skill.Installed
	if allRequested {
		targets, err := collectRemovalTargets(positionals, tools, normalizedScope, cwd)
		if err != nil {
			fmt.Fprintf(a.errOut, "uninstall failed: %v\n", err)
			return 1
		}
		if len(targets) == 0 {
			fmt.Fprintln(a.out, "no matching skills found")
			return 0
		}
		if !confirmRemoval(a.out, targets) {
			fmt.Fprintln(a.out, "canceled")
			return 0
		}
	}
	if len(positionals) == 0 && !allRequested {
		fmt.Fprintln(a.errOut, "uninstall requires a skill name (or use -a)")
		return 2
	}
	err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
		var uninstallErr error
		if len(positionals) == 0 {
			records, uninstallErr = skill.UninstallAll(normalizedScope, cwd, tools)
			return uninstallErr
		}
		name := positionals[0]
		records, uninstallErr = skill.Uninstall(name, normalizedScope, cwd, tools, *forceShort || *forceLong)
		return uninstallErr
	})
	if err != nil {
		fmt.Fprintf(a.errOut, "uninstall failed: %v\n", err)
		return 1
	}

	for _, record := range records {
		fmt.Fprintf(a.out, "removed %s from %s (%s)\n", record.Name, record.Path, record.Client)
	}
	return 0
}

func (a *App) printUninstallHelp() {
	fmt.Fprintf(a.out, `Usage: %s uninstall [name] [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>] [--all|-a]

Examples:
  %s uninstall my-skill -l -c opencode
  %s rm my-skill -g -a
  %s uninstall -g -a
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

func collectRemovalTargets(positionals []string, tools []installer.Tool, scope, cwd string) ([]skill.Installed, error) {
	items, err := skill.List([]string{scope}, cwd, tools)
	if err != nil {
		return nil, err
	}

	var nameFilter string
	if len(positionals) > 0 {
		nameFilter = positionals[0]
	}

	var targets []skill.Installed
	for _, item := range items {
		if nameFilter != "" && item.Name != nameFilter {
			continue
		}
		targets = append(targets, item)
	}
	return targets, nil
}

func confirmRemoval(out io.Writer, targets []skill.Installed) bool {
	fmt.Fprintf(out, "About to remove %d skill(s):\n", len(targets))
	for _, item := range targets {
		fmt.Fprintf(out, "- %s (%s/%s)\n", item.Name, item.Client, item.Scope)
	}
	return confirmPrompt(out, "Type 'yes' to continue: ")
}
