package skillcli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/skill"
)

func (a *App) runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	scope := fs.String("scope", "", "deprecated: use --global or --local")
	globalShort := fs.Bool("g", false, "install to user/global scope")
	globalLong := fs.Bool("global", false, "install to user/global scope")
	localShort := fs.Bool("l", false, "install to project/local scope")
	localLong := fs.Bool("local", false, "install to project/local scope")
	projectLong := fs.Bool("project", false, "install to project/local scope")
	forceShort := fs.Bool("f", false, "overwrite existing skills")
	forceLong := fs.Bool("force", false, "overwrite existing skills")
	allShort := fs.Bool("a", false, "install for all clients")
	allLong := fs.Bool("all", false, "install for all clients")
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
		a.printInstallHelp()
		return 0
	}

	if len(positionals) == 0 {
		fmt.Fprintln(a.errOut, "install requires a repo, path, or local skill name")
		return 2
	}

	clientValue, err := resolveClientValue(*clientFlag, *clientShort, *toolFlag, *allShort || *allLong)
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
	source := positionals[0]
	cwd, _ := os.Getwd()
	force := *forceShort || *forceLong

	var records []skill.Installed
	err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), time.Second, func() error {
		var installErr error
		records, installErr = skill.Install(source, normalizedScope, cwd, tools, force)
		return installErr
	})
	if err != nil && !force && isAlreadyExistsError(err) {
		if !confirmPrompt(a.out, "Skill already exists. Overwrite? Type 'yes' to continue: ") {
			fmt.Fprintln(a.out, "canceled")
			return 0
		}
		err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), time.Second, func() error {
			var installErr error
			records, installErr = skill.Install(source, normalizedScope, cwd, tools, true)
			return installErr
		})
	}
	if err != nil {
		fmt.Fprintf(a.errOut, "install failed: %v\n", err)
		return 1
	}

	for _, record := range records {
		fmt.Fprintf(a.out, "installed %s -> %s (%s)\n", record.Name, record.Path, record.Client)
	}
	return 0
}

func (a *App) printInstallHelp() {
	fmt.Fprintf(a.out, `Usage: %s install <repo|path|name> [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>] [--all|-a]

Examples:
  %s install openai/skills
  %s install D:\downloads\agent-skills -c opencode
  %s install react-best-practices -c opencode
  %s i https://github.com/openai/skills.git -c codex,claude
  %s install openai/skills -g -c opencode
  %s install openai/skills -g -a
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "skill already exists:")
}
