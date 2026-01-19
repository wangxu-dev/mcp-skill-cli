package mcpcli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
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

func (a *App) runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	scope := fs.String("scope", "", "deprecated: use --global or --local")
	globalShort := fs.Bool("g", false, "install to user/global scope")
	globalLong := fs.Bool("global", false, "install to user/global scope")
	localShort := fs.Bool("l", false, "install to project/local scope")
	localLong := fs.Bool("local", false, "install to project/local scope")
	projectLong := fs.Bool("project", false, "install to project/local scope")
	forceShort := fs.Bool("f", false, "overwrite existing servers")
	forceLong := fs.Bool("force", false, "overwrite existing servers")
	allShort := fs.Bool("a", false, "install for all clients")
	allLong := fs.Bool("all", false, "install for all clients")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode")
	clientShort := fs.String("c", "", "alias for --client")
	toolFlag := fs.String("tool", "", "deprecated: use --client")
	nameFlag := fs.String("name", "", "server name (for inline definition)")
	transportFlag := fs.String("transport", "", "transport: http or stdio")
	urlFlag := fs.String("url", "", "server URL for http transport")
	commandFlag := fs.String("command", "", "command for stdio transport")
	argsFlag := fs.String("args", "", "comma-separated args for stdio transport")
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

	clientValue, err := resolveClientValue(*clientFlag, *clientShort, *toolFlag, *allShort || *allLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	clients, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}
	clients, err = normalizeMcpClients(clients, clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	clients, err = normalizeMcpClients(clients, clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	clients, err = normalizeMcpClients(clients, clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}

	normalizedScope, err := resolveScope(*scope, *globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid scope: %v\n", err)
		return 2
	}
	if normalizedScope == installer.ScopeProject && containsClient(clients, installer.ToolCodex) {
		fmt.Fprintln(a.errOut, "codex MCP only supports user scope; use --global")
		return 2
	}
	if normalizedScope == installer.ScopeProject && containsClient(clients, installer.ToolCodex) {
		fmt.Fprintln(a.errOut, "codex MCP only supports user scope; use --global")
		return 2
	}

	var def mcp.Definition
	var records []mcp.Installed
	cwd, _ := os.Getwd()
	force := *forceShort || *forceLong

	if usesInlineDefinition(*nameFlag, *transportFlag, *urlFlag, *commandFlag, *argsFlag) {
		args := splitArgsCSV(*argsFlag)
		def, err = mcp.DefinitionFromArgs(*nameFlag, *transportFlag, *urlFlag, *commandFlag, args)
		if err != nil {
			fmt.Fprintf(a.errOut, "invalid definition: %v\n", err)
			return 2
		}
	} else {
		if len(positionals) == 0 {
			fmt.Fprintln(a.errOut, "install requires a name or path")
			return 2
		}
		source := positionals[0]
		if fileExists(source) {
			def, err = mcp.LoadDefinitionFromFile(source)
			if err != nil {
				fmt.Fprintf(a.errOut, "install failed: %v\n", err)
				return 1
			}
		} else {
			if err := registryindex.EnsureIndexes(); err == nil {
				entry, ok, err := registryindex.FindMCP(source)
				if err != nil {
					fmt.Fprintf(a.errOut, "install failed: %v\n", err)
					return 1
				}
				if ok {
					records, err = installFromRegistryEntry(entry, registryInstallOptions{
						Scope:   normalizedScope,
						Cwd:     cwd,
						Clients: clients,
						Force:   force,
						Out:     bufio.NewWriter(a.out),
						ErrOut:  bufio.NewWriter(a.errOut),
					})
					if err != nil {
						fmt.Fprintf(a.errOut, "install failed: %v\n", err)
						return 1
					}
					for _, record := range records {
						fmt.Fprintf(a.out, "installed %s -> %s (%s)\n", record.Name, record.Path, record.Client)
					}
					return 0
				}
			}
			def, err = mcp.LoadLocalDefinition(source)
			if err != nil {
				fmt.Fprintf(a.errOut, "install failed: server not found in registry or local store: %s\n", source)
				return 1
			}
		}
	}

	if _, err := mcp.SaveLocalDefinition(def); err != nil {
		fmt.Fprintf(a.errOut, "install failed: %v\n", err)
		return 1
	}

	spinner := cli.StartSpinner(a.errOut, "")
	records, err = mcp.Install(def, normalizedScope, cwd, clients, force)
	spinner.Stop()
	if err != nil && !force && isAlreadyExistsError(err) {
		if !confirmPrompt(a.out, "Server already exists. Overwrite? Type 'yes' to continue: ") {
			fmt.Fprintln(a.out, "canceled")
			return 0
		}
		spinner = cli.StartSpinner(a.errOut, "")
		records, err = mcp.Install(def, normalizedScope, cwd, clients, true)
		spinner.Stop()
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
		fmt.Fprintln(a.errOut, "list accepts at most one server name")
		return 2
	}
	nameFilter := ""
	if len(positionals) == 1 {
		nameFilter = positionals[0]
	}

	clientValue, err := resolveListClientValue(*clientFlag, *clientShort, *toolFlag)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	clients, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}

	scopes := resolveListScopes(*globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if containsScope(scopes, installer.ScopeProject) && containsClient(clients, installer.ToolCodex) {
		fmt.Fprintln(a.errOut, "codex MCP only supports user scope; use --global")
		return 2
	}
	cwd, _ := os.Getwd()
	items, err := mcp.List(scopes, cwd, clients)
	if err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}

	if len(items) == 0 {
		fmt.Fprintln(a.out, "no servers installed")
		return 0
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Client != items[j].Client {
			return items[i].Client < items[j].Client
		}
		if items[i].Scope != items[j].Scope {
			return items[i].Scope < items[j].Scope
		}
		return items[i].Name < items[j].Name
	})

	var (
		lastClient installer.Tool
		lastScope  string
		writer     *tabwriter.Writer
		matched    int
		printed    bool
	)

	for _, item := range items {
		if !matchesFilter(item.Name, nameFilter) {
			continue
		}
		if writer == nil || item.Client != lastClient || item.Scope != lastScope {
			if writer != nil {
				if err := writer.Flush(); err != nil {
					fmt.Fprintf(a.errOut, "list failed: %v\n", err)
					return 1
				}
			}
			if printed {
				fmt.Fprintln(a.out)
			}
			fmt.Fprintf(a.out, "%s (%s)\n", item.Client, item.Scope)
			writer = tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(writer, "NAME\tTRANSPORT\tPATH")
			printed = true
			lastClient = item.Client
			lastScope = item.Scope
		}
		matched++
		fmt.Fprintf(writer, "%s\t%s\t%s\n", item.Name, displayTransport(item.Transport), item.Path)
	}

	if matched == 0 {
		fmt.Fprintln(a.out, "no matching servers found")
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

func (a *App) runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	scope := fs.String("scope", "", "deprecated: use --global or --local")
	globalShort := fs.Bool("g", false, "remove from user/global scope")
	globalLong := fs.Bool("global", false, "remove from user/global scope")
	localShort := fs.Bool("l", false, "remove from project/local scope")
	localLong := fs.Bool("local", false, "remove from project/local scope")
	projectLong := fs.Bool("project", false, "remove from project/local scope")
	forceShort := fs.Bool("f", false, "ignore missing servers")
	forceLong := fs.Bool("force", false, "ignore missing servers")
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
	clients, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}

	normalizedScope, err := resolveScope(*scope, *globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid scope: %v\n", err)
		return 2
	}

	if allRequested {
		targets, err := collectRemovalTargets(positionals, normalizedScope, clients)
		if err != nil {
			fmt.Fprintf(a.errOut, "uninstall failed: %v\n", err)
			return 1
		}
		if len(targets) == 0 {
			fmt.Fprintln(a.out, "no matching servers found")
			return 0
		}
		if !confirmRemoval(a.out, targets) {
			fmt.Fprintln(a.out, "canceled")
			return 0
		}
	}

	cwd, _ := os.Getwd()
	var records []mcp.Installed
	if len(positionals) == 0 {
		if !allRequested {
			fmt.Fprintln(a.errOut, "uninstall requires a server name (or use -a)")
			return 2
		}
		records, err = mcp.UninstallAll(normalizedScope, cwd, clients)
	} else {
		name := positionals[0]
		records, err = mcp.Uninstall(name, normalizedScope, cwd, clients, *forceShort || *forceLong)
	}
	if err != nil {
		fmt.Fprintf(a.errOut, "uninstall failed: %v\n", err)
		return 1
	}

	for _, record := range records {
		fmt.Fprintf(a.out, "removed %s from %s (%s)\n", record.Name, record.Path, record.Client)
	}
	return 0
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

func (a *App) printHelp() {
	fmt.Fprintf(a.out, `Usage: %s <command> [options]

Commands:
  install|i <source>   Install MCP servers from registry, local store, or file
  list                 List installed MCP servers
  update|upgrade        Update installed MCP servers from registry
  uninstall|remove|rm  Remove installed MCP servers
  clean                Clear local store (~/.mcp-skill/skill and ~/.mcp-skill/mcp)

Use "%s <command> -h" for command help.
`, a.binaryName, a.binaryName)
}

func (a *App) printInstallHelp() {
	fmt.Fprintf(a.out, `Usage: %s install <name|path> [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>] [--all|-a]
       %s install --name <name> --transport <http|stdio> [--url <url> | --command <cmd>] [--args <a,b>] [--client|-c <list>] [--all|-a]

Examples:
  %s install github -c claude
  %s install D:\mcp\github.json -c codex
  %s install --name github --transport http --url https://example.com/mcp -c claude
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

func (a *App) printListHelp() {
	fmt.Fprintf(a.out, `Usage: %s list [name] [--global|-g] [--local|-l] [--client|-c <list>]

Examples:
  %s list
  %s list github -g -c claude
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

func (a *App) runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	globalShort := fs.Bool("g", false, "update global/user scope")
	globalLong := fs.Bool("global", false, "update global/user scope")
	localShort := fs.Bool("l", false, "update local/project scope")
	localLong := fs.Bool("local", false, "update local/project scope")
	projectLong := fs.Bool("project", false, "update local/project scope")
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
		a.printUpdateHelp()
		return 0
	}
	if len(positionals) > 1 {
		fmt.Fprintln(a.errOut, "update accepts at most one server name")
		return 2
	}
	nameFilter := ""
	if len(positionals) == 1 {
		nameFilter = positionals[0]
	}

	clientValue, err := resolveListClientValue(*clientFlag, *clientShort, *toolFlag)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	clients, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}
	clients, err = normalizeMcpClients(clients, clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client selection: %v\n", err)
		return 2
	}
	scopes := resolveListScopes(*globalShort || *globalLong, *localShort || *localLong || *projectLong)
	if containsScope(scopes, installer.ScopeProject) && containsClient(clients, installer.ToolCodex) {
		fmt.Fprintln(a.errOut, "codex MCP only supports user scope; use --global")
		return 2
	}
	cwd, _ := os.Getwd()
	items, err := mcp.List(scopes, cwd, clients)
	if err != nil {
		fmt.Fprintf(a.errOut, "update failed: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(a.out, "no servers installed")
		return 0
	}

	var targets []mcp.Installed
	for _, item := range items {
		if nameFilter != "" && item.Name != nameFilter {
			continue
		}
		targets = append(targets, item)
	}
	if len(targets) == 0 {
		fmt.Fprintln(a.out, "no matching servers found")
		return 0
	}

	spinner := cli.StartSpinner(a.errOut, "")
	if err := registryindex.EnsureIndexes(); err != nil {
		spinner.Stop()
		fmt.Fprintf(a.errOut, "update failed: %v\n", err)
		return 1
	}
	spinner.Stop()

	type result struct {
		item    mcp.Installed
		message string
		err     error
	}
	var results []result
	for _, item := range targets {
		entry, ok, err := registryindex.FindMCP(item.Name)
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		if !ok {
			results = append(results, result{item: item, message: "not in registry"})
			continue
		}

		record, ok, err := registryindex.LocalRecordFor("mcp", item.Name)
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		if ok && entry.Head != "" && record.Head == entry.Head {
			results = append(results, result{item: item, message: "already latest"})
			continue
		}

		_, err = installFromRegistryEntry(entry, registryInstallOptions{
			Scope:   item.Scope,
			Cwd:     cwd,
			Clients: []installer.Tool{item.Client},
			Force:   true,
			Out:     bufio.NewWriter(a.out),
			ErrOut:  bufio.NewWriter(a.errOut),
		})
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		results = append(results, result{item: item, message: "updated"})
	}
	for _, res := range results {
		if res.err != nil {
			fmt.Fprintf(a.errOut, "update failed for %s (%s/%s): %v\n", res.item.Name, res.item.Client, res.item.Scope, res.err)
			continue
		}
		fmt.Fprintf(a.out, "%s (%s/%s): %s\n", res.item.Name, res.item.Client, res.item.Scope, res.message)
	}
	return 0
}

func (a *App) printUpdateHelp() {
	fmt.Fprintf(a.out, `Usage: %s update [name] [--global|-g] [--local|-l] [--client|-c <list>]

Examples:
  %s update
  %s update github -g -c claude
`, a.binaryName, a.binaryName, a.binaryName)
}

func (a *App) printUninstallHelp() {
	fmt.Fprintf(a.out, `Usage: %s uninstall [name] [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>] [--all|-a]

Examples:
  %s uninstall github -l -c claude
  %s rm github -g -a
  %s rm -g -a
`, a.binaryName, a.binaryName, a.binaryName)
}

func (a *App) printCleanHelp() {
	fmt.Fprintf(a.out, `Usage: %s clean

Clears:
  ~/.mcp-skill/skill
  ~/.mcp-skill/mcp
`, a.binaryName)
}

func usesInlineDefinition(name, transport, url, command, args string) bool {
	return strings.TrimSpace(name) != "" ||
		strings.TrimSpace(transport) != "" ||
		strings.TrimSpace(url) != "" ||
		strings.TrimSpace(command) != "" ||
		strings.TrimSpace(args) != ""
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

func splitArgs(args []string) ([]string, []string) {
	var flags []string
	var positionals []string
	valueFlags := map[string]bool{
		"--scope":     true,
		"--client":    true,
		"--tool":      true,
		"-c":          true,
		"--name":      true,
		"--transport": true,
		"--url":       true,
		"--command":   true,
		"--args":      true,
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

func splitArgsCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func matchesFilter(name, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

func displayTransport(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "server already exists:")
}

func collectRemovalTargets(positionals []string, scope string, clients []installer.Tool) ([]mcp.Installed, error) {
	cwd, _ := os.Getwd()
	items, err := mcp.List([]string{scope}, cwd, clients)
	if err != nil {
		return nil, err
	}

	var nameFilter string
	if len(positionals) > 0 {
		nameFilter = positionals[0]
	}

	var targets []mcp.Installed
	for _, item := range items {
		if nameFilter != "" && item.Name != nameFilter {
			continue
		}
		targets = append(targets, item)
	}
	return targets, nil
}

func confirmRemoval(out io.Writer, targets []mcp.Installed) bool {
	fmt.Fprintf(out, "About to remove %d server(s):\n", len(targets))
	for _, item := range targets {
		fmt.Fprintf(out, "- %s (%s/%s)\n", item.Name, item.Client, item.Scope)
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

func isHelp(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "-h" || value == "--help" || value == "help"
}

func normalizeMcpClients(clients []installer.Tool, clientValue string) ([]installer.Tool, error) {
	allowed := map[installer.Tool]bool{
		installer.ToolClaude:   true,
		installer.ToolCodex:    true,
		installer.ToolGemini:   true,
		installer.ToolOpenCode: true,
	}
	var filtered []installer.Tool
	var unsupported []string
	for _, client := range clients {
		if allowed[client] {
			filtered = append(filtered, client)
			continue
		}
		unsupported = append(unsupported, string(client))
	}
	if len(unsupported) > 0 && strings.TrimSpace(clientValue) != "" && clientValue != "all" {
		return nil, fmt.Errorf("unsupported MCP clients: %s", strings.Join(unsupported, ", "))
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no supported MCP clients selected")
	}
	return filtered, nil
}

func containsClient(clients []installer.Tool, target installer.Tool) bool {
	for _, client := range clients {
		if client == target {
			return true
		}
	}
	return false
}

func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
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

func resolveDefinition(source string) (mcp.Definition, error) {
	if fileExists(source) {
		return mcp.LoadDefinitionFromFile(source)
	}

	if err := registryindex.EnsureIndexes(); err != nil {
		def, localErr := mcp.LoadLocalDefinition(source)
		if localErr != nil {
			return mcp.Definition{}, err
		}
		return def, nil
	}

	entry, ok, err := registryindex.FindMCP(source)
	if err != nil {
		return mcp.Definition{}, err
	}
	if ok {
		entryType := normalizeEntryType(entry)
		if entryType != "http" {
			return mcp.Definition{}, fmt.Errorf("stdio entries must be installed from the registry")
		}
		if err := registryindex.SyncMCP(entry); err != nil {
			def, localErr := mcp.LoadLocalDefinition(entry.Name)
			if localErr != nil {
				return mcp.Definition{}, err
			}
			return def, nil
		}
		return mcp.LoadLocalDefinition(entry.Name)
	}

	def, err := mcp.LoadLocalDefinition(source)
	if err != nil {
		return mcp.Definition{}, fmt.Errorf("server not found in registry or local store: %s", source)
	}
	return def, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
