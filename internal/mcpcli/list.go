package mcpcli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
)

func (a *App) runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	globalShort := fs.Bool("g", false, "show global/user scope")
	globalLong := fs.Bool("global", false, "show global/user scope")
	localShort := fs.Bool("l", false, "show local/project scope")
	localLong := fs.Bool("local", false, "show local/project scope")
	projectLong := fs.Bool("project", false, "show local/project scope")
	availableShort := fs.Bool("a", false, "show available servers in registry")
	availableLong := fs.Bool("available", false, "show available servers in registry")
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

	if *availableShort || *availableLong {
		return a.runListAvailable(nameFilter)
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
	var items []mcp.Installed
	err = cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
		var listErr error
		items, listErr = mcp.List(scopes, cwd, clients)
		return listErr
	})
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

func (a *App) runListAvailable(nameFilter string) int {
	var outputErr error
	type row struct {
		name        string
		typ         string
		updatedAt   string
		description string
	}
	var rows []row
	err := cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), cli.DefaultSpinnerDelay, func() error {
		if err := registryindex.EnsureIndexes(); err != nil {
			outputErr = err
			return err
		}
		index, err := registryindex.LoadMCPIndex()
		if err != nil {
			outputErr = err
			return err
		}
		entries := index.MCP
		if len(entries) == 0 {
			entries = index.Servers
		}
		if len(entries) == 0 {
			return nil
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})

		for _, entry := range entries {
			if !matchesFilter(entry.Name, nameFilter) {
				continue
			}
			rows = append(rows, row{
				name:        entry.Name,
				typ:         displayTransport(entry.Type),
				updatedAt:   entry.UpdatedAt,
				description: truncateDescription(entry.Description, 80),
			})
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", outputErr)
		return 1
	}

	if len(rows) == 0 {
		fmt.Fprintln(a.out, "no matching servers found")
		return 0
	}
	writer := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tTYPE\tUPDATED\tDESCRIPTION")
	for _, item := range rows {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", item.name, item.typ, item.updatedAt, item.description)
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) printListHelp() {
	fmt.Fprintf(a.out, `Usage: %s list [name] [--available|-a] [--global|-g] [--local|-l] [--client|-c <list>]

What it does:
  - Default: list installed MCP servers
  - With --available: list registry MCP servers

Examples:
  %s list
  %s list github -g -c claude
  %s list --available
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}
