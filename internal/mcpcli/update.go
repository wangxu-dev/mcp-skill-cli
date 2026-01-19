package mcpcli

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
)

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

	err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
		return registryindex.EnsureIndexes()
	})
	if err != nil {
		fmt.Fprintf(a.errOut, "update failed: %v\n", err)
		return 1
	}

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

		needsUpdate, err := needsMcpUpdate(entry)
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		if !needsUpdate {
			results = append(results, result{item: item, message: "already latest"})
			continue
		}

		_, err = installFromRegistryEntry(entry, registryInstallOptions{
			Scope:      item.Scope,
			Cwd:        cwd,
			Clients:    []installer.Tool{item.Client},
			Force:      true,
			Out:        bufio.NewWriter(a.out),
			ErrOut:     bufio.NewWriter(a.errOut),
			SpinnerOut: a.errOut,
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

What it does:
  - Checks registry for changes and reinstalls when needed
  - If name is omitted, updates all installed servers in the selected scope/clients

Examples:
  %s update
  %s update github -g -c claude
`, a.binaryName, a.binaryName, a.binaryName)
}
