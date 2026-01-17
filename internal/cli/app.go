package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

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
	case "list":
		return a.runList(args[1:])
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
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode")
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
	records, err := installer.InstallFromInput(source, normalizedScope, tools, cwd, *forceShort || *forceLong)
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
  install|i <source>   Install skills from repo, local path, or local store
  list               List installed skills
  uninstall|remove|rm <name>   Remove an installed skill
  clean              Clear local store (~/.mcp-skill/skill and ~/.mcp-skill/mcp)

Use "%s <command> -h" for command help.
`, a.binaryName, a.binaryName)
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

func (a *App) runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	globalShort := fs.Bool("g", false, "show global/user scope")
	globalLong := fs.Bool("global", false, "show global/user scope")
	localShort := fs.Bool("l", false, "show local/project scope")
	localLong := fs.Bool("local", false, "show local/project scope")
	projectLong := fs.Bool("project", false, "show local/project scope")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode")
	clientShort := fs.String("c", "", "alias for --client")
	toolFlag := fs.String("tool", "", "deprecated: use --client")
	helpShort := fs.Bool("h", false, "show help")
	helpLong := fs.Bool("help", false, "show help")

	flags, positionals := splitArgs(args)
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if *helpShort || *helpLong {
		a.printListHelp()
		return 0
	}

	if len(positionals) > 1 {
		fmt.Fprintln(a.errOut, "list accepts at most one skill name")
		return 2
	}
	skillFilter := ""
	if len(positionals) == 1 {
		skillFilter = positionals[0]
	}

	clientValue, err := resolveListClientValue(*clientFlag, *clientShort, *toolFlag)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	tools, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}

	scopes := resolveListScopes(*globalShort || *globalLong, *localShort || *localLong || *projectLong)
	cwd, _ := os.Getwd()
	items, err := installer.ListInstalled(tools, scopes, cwd)
	if err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}

	if len(items) == 0 {
		fmt.Fprintln(a.out, "no skills installed")
		return 0
	}

	var (
		lastTool  installer.Tool
		lastScope string
		matched   int
		printed   bool
		writer    *tabwriter.Writer
	)

	for _, item := range items {
		if !matchesSkillFilter(item.SkillName, skillFilter) {
			continue
		}
		if writer == nil || item.Tool != lastTool || item.Scope != lastScope {
			if writer != nil {
				if err := writer.Flush(); err != nil {
					fmt.Fprintf(a.errOut, "list failed: %v\n", err)
					return 1
				}
			}
			if printed {
				fmt.Fprintln(a.out)
			}
			fmt.Fprintf(a.out, "%s (%s)\n", item.Tool, item.Scope)
			writer = tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(writer, "SKILL\tPATH")
			printed = true
			lastTool = item.Tool
			lastScope = item.Scope
		}
		matched++
		fmt.Fprintf(writer, "%s\t%s\n", item.SkillName, item.Path)
	}

	if matched == 0 {
		fmt.Fprintln(a.out, "no matching skills found")
		return 0
	}

	if writer != nil {
		if err := writer.Flush(); err != nil {
			fmt.Fprintf(a.errOut, "list failed: %v\n", err)
			return 1
		}
	}
	return 0
}

func (a *App) printListHelp() {
	fmt.Fprintf(a.out, `Usage: %s list [skill] [--global|-g] [--local|-l] [--client|-c <list>]

Examples:
  %s list
  %s list my-skill -l
  %s list my-skill -g -c opencode
  %s list -g
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

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
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode")
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
	var records []installer.RemoveRecord
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
	if len(positionals) == 0 {
		records, err = installer.UninstallAll(normalizedScope, tools, cwd)
	} else {
		name := positionals[0]
		records, err = installer.UninstallSkill(name, normalizedScope, tools, cwd, *forceShort || *forceLong)
	}
	if err != nil {
		fmt.Fprintf(a.errOut, "uninstall failed: %v\n", err)
		return 1
	}

	for _, record := range records {
		fmt.Fprintf(a.out, "removed %s from %s (%s)\n", record.SkillName, record.Path, record.Tool)
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

	skillRoot, err := installer.LocalSkillStore()
	if err != nil {
		fmt.Fprintf(a.errOut, "clean failed: %v\n", err)
		return 1
	}
	mcpRoot, err := installer.LocalMcpStore()
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

	if err := installer.CleanLocalStore(); err != nil {
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

func resolveListScopes(global bool, local bool) []string {
	if global && local {
		return []string{installer.ScopeUser, installer.ScopeProject}
	}
	if global {
		return []string{installer.ScopeUser}
	}
	if local {
		return []string{installer.ScopeProject}
	}
	return []string{installer.ScopeProject}
}

func collectRemovalTargets(positionals []string, tools []installer.Tool, scope, cwd string) ([]installer.RemoveRecord, error) {
	items, err := installer.ListInstalled(tools, []string{scope}, cwd)
	if err != nil {
		return nil, err
	}

	var nameFilter string
	if len(positionals) > 0 {
		nameFilter = positionals[0]
	}

	var targets []installer.RemoveRecord
	for _, item := range items {
		if nameFilter != "" && item.SkillName != nameFilter {
			continue
		}
		targets = append(targets, installer.RemoveRecord{
			SkillName: item.SkillName,
			Tool:      item.Tool,
			Scope:     item.Scope,
			Path:      item.Path,
		})
	}
	return targets, nil
}

func confirmRemoval(out io.Writer, targets []installer.RemoveRecord) bool {
	fmt.Fprintf(out, "About to remove %d skill(s):\n", len(targets))
	for _, item := range targets {
		fmt.Fprintf(out, "- %s (%s/%s)\n", item.SkillName, item.Tool, item.Scope)
	}
	return confirmPrompt(out, "Type 'yes' to continue: ")
}

func confirmPrompt(out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "yes"
}

func countEntries(path string) (int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return len(entries), nil
}

func resolveClientValue(clientFlag, clientShort, toolFlag string, all bool) (string, error) {
	if all {
		if clientFlag != "" || clientShort != "" || toolFlag != "" {
			return "", fmt.Errorf("cannot combine --all with --client")
		}
		return "all", nil
	}

	clientValue := clientFlag
	if clientShort != "" {
		clientValue = clientShort
	}
	if toolFlag != "" {
		if clientValue != "" && clientValue != toolFlag {
			return "", fmt.Errorf("conflicting client flags: use only one of --client/-c/--tool")
		}
		clientValue = toolFlag
	}
	if strings.TrimSpace(clientValue) == "" {
		return "", fmt.Errorf("choose a client with --client/-c or use --all")
	}
	return clientValue, nil
}

func resolveListClientValue(clientFlag, clientShort, toolFlag string) (string, error) {
	clientValue := clientFlag
	if clientShort != "" {
		clientValue = clientShort
	}
	if toolFlag != "" {
		if clientValue != "" && clientValue != toolFlag {
			return "", fmt.Errorf("conflicting client flags: use only one of --client/-c/--tool")
		}
		clientValue = toolFlag
	}
	if strings.TrimSpace(clientValue) == "" {
		return "all", nil
	}
	return clientValue, nil
}

func matchesSkillFilter(name, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
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
