package mcpcli

import (
	"bufio"
	"flag"
	"fmt"
	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
	"os"
	"strings"
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
						Scope:      normalizedScope,
						Cwd:        cwd,
						Clients:    clients,
						Force:      force,
						Out:        bufio.NewWriter(a.out),
						ErrOut:     bufio.NewWriter(a.errOut),
						SpinnerOut: a.errOut,
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

	err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
		var installErr error
		records, installErr = mcp.Install(def, normalizedScope, cwd, clients, force)
		return installErr
	})
	if err != nil && !force && isAlreadyExistsError(err) {
		if !confirmPrompt(a.out, "Server already exists. Overwrite? Type 'yes' to continue: ") {
			fmt.Fprintln(a.out, "canceled")
			return 0
		}
		err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
			var installErr error
			records, installErr = mcp.Install(def, normalizedScope, cwd, clients, true)
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
	fmt.Fprintf(a.out, `Usage: %s install <name|path> [--global|-g] [--local|-l] [--force|-f] [--client|-c <list>] [--all|-a]
       %s install --name <name> --transport <http|stdio> [--url <url> | --command <cmd>] [--args <a,b>] [--client|-c <list>] [--all|-a]

What it does:
  - Registry name: checks requirements, prompts for inputs, clones/builds if needed, then writes config
  - File path: loads the MCP definition JSON and writes config
  - Inline definition: uses flags to build a definition and writes config

Examples:
  %s install github -c claude
  %s install D:\mcp\github.json -c codex
  %s install --name github --transport http --url https://example.com/mcp -c claude
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}

func usesInlineDefinition(name, transport, url, command, args string) bool {
	return strings.TrimSpace(name) != "" ||
		strings.TrimSpace(transport) != "" ||
		strings.TrimSpace(url) != "" ||
		strings.TrimSpace(command) != "" ||
		strings.TrimSpace(args) != ""
}
