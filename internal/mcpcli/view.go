package mcpcli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
)

func (a *App) runView(args []string) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	installedFlag := fs.Bool("installed", false, "show installed server info")
	globalShort := fs.Bool("g", false, "show global/user scope (installed only)")
	globalLong := fs.Bool("global", false, "show global/user scope (installed only)")
	localShort := fs.Bool("l", false, "show local/project scope (installed only)")
	localLong := fs.Bool("local", false, "show local/project scope (installed only)")
	projectLong := fs.Bool("project", false, "show local/project scope (installed only)")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode (installed only)")
	clientShort := fs.String("c", "", "alias for --client")
	toolFlag := fs.String("tool", "", "deprecated: use --client")
	helpShort := fs.Bool("h", false, "show help")
	helpLong := fs.Bool("help", false, "show help")

	flags, positionals := splitArgs(args)
	if err := fs.Parse(flags); err != nil {
		return 2
	}
	if *helpShort || *helpLong {
		a.printViewHelp()
		return 0
	}
	if len(positionals) == 0 {
		fmt.Fprintln(a.errOut, "view requires a server name")
		return 2
	}
	if len(positionals) > 1 {
		fmt.Fprintln(a.errOut, "view accepts a single server name")
		return 2
	}

	name := positionals[0]
	if *installedFlag {
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
			fmt.Fprintf(a.errOut, "view failed: %v\n", err)
			return 1
		}

		var matches []mcp.Installed
		for _, item := range items {
			if item.Name == name {
				matches = append(matches, item)
			}
		}
		if len(matches) == 0 {
			fmt.Fprintln(a.out, "no matching servers found")
			return 0
		}
		def, defErr := mcp.LoadLocalDefinition(name)
		for idx, item := range matches {
			if idx > 0 {
				fmt.Fprintln(a.out)
			}
			fmt.Fprintf(a.out, "%s (%s)\n", item.Client, item.Scope)
			fmt.Fprintf(a.out, "path: %s\n", item.Path)
			fmt.Fprintf(a.out, "transport: %s\n", displayTransport(item.Transport))
			if defErr == nil {
				printDefinition(a.out, def)
			}
		}
		return 0
	}

	err := cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), time.Second, func() error {
		return registryindex.EnsureIndexes()
	})
	if err != nil {
		fmt.Fprintf(a.errOut, "view failed: %v\n", err)
		return 1
	}
	entry, ok, err := registryindex.FindMCP(name)
	if err != nil {
		fmt.Fprintf(a.errOut, "view failed: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(a.out, "server not found in registry")
		return 0
	}
	printMcpEntry(a.out, entry)
	return 0
}

func (a *App) printViewHelp() {
	fmt.Fprintf(a.out, `Usage: %s view <name> [--installed] [--global|-g] [--local|-l] [--client|-c <list>]

What it does:
  - Default: show registry metadata for a server
  - With --installed: show installed server details and local definition

Examples:
  %s view context7
  %s view context7 --installed -g -c claude
`, a.binaryName, a.binaryName, a.binaryName)
}
