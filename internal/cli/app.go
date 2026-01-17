package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"mcp-skill-manager/internal/installer"
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
	default:
		fmt.Fprintf(a.errOut, "unknown command: %s\n", args[0])
		a.printHelp()
		return 2
	}
}

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
	clientFlag := fs.String("client", "all", "comma-separated clients: claude,codex,gemini,opencode")
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
		fmt.Fprintln(a.errOut, "install requires a GitHub repo (owner/repo or URL)")
		return 2
	}

	clientValue := *clientFlag
	if *clientShort != "" {
		clientValue = *clientShort
	}
	if *toolFlag != "" {
		if clientValue != "all" && clientValue != *toolFlag {
			fmt.Fprintln(a.errOut, "conflicting client flags: use only one of --client/-c/--tool")
			return 2
		}
		clientValue = *toolFlag
	}

	tools, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid tool list: %v\n", err)
		return 2
	}

	normalizedScope, err := resolveScope(*scope, *globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid scope: %v\n", err)
		return 2
	}
	repo := positionals[0]
	cwd, _ := os.Getwd()
	records, err := installer.InstallFromRepo(repo, normalizedScope, tools, cwd, *forceShort || *forceLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "install failed: %v\n", err)
		return 1
	}

	for _, record := range records {
		fmt.Fprintf(a.out, "installed %s -> %s (%s)\n", record.SkillName, record.DestPath, record.Tool)
	}
	return 0
}

func (a *App) printHelp() {
	fmt.Fprintf(a.out, `Usage: %s <command> [options]

Commands:
  install|i <repo>   Install skills from a GitHub repository

Use "%s install -h" for command help.
`, a.binaryName, a.binaryName)
}

func (a *App) printInstallHelp() {
	fmt.Fprintf(a.out, `Usage: %s install <repo> [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>]

Examples:
  %s install openai/skills
  %s i https://github.com/openai/skills.git -c codex,claude
  %s install openai/skills -g -c opencode
`, a.binaryName, a.binaryName, a.binaryName)
}

func isHelp(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "-h" || value == "--help" || value == "help"
}

func resolveScope(scope string, global bool, local bool) (string, error) {
	if global && local {
		return "", fmt.Errorf("choose only one of global or local")
	}
	if global {
		return installer.ScopeUser, nil
	}
	if local {
		return installer.ScopeProject, nil
	}

	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case "", "local", "project":
		return installer.ScopeProject, nil
	case "global", "user":
		return installer.ScopeUser, nil
	default:
		return "", fmt.Errorf("unknown scope: %s", scope)
	}
}

func splitArgs(args []string) ([]string, []string) {
	var flags []string
	var positionals []string
	valueFlags := map[string]bool{
		"--scope":  true,
		"--tool":   true,
		"--client": true,
		"-c":       true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if valueFlags[arg] && !strings.Contains(arg, "=") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
			continue
		}
		positionals = append(positionals, arg)
	}

	return flags, positionals
}
