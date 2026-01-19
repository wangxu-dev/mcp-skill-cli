package skillcli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/registryindex"
	"mcp-skill-manager/internal/skill"
)

func (a *App) runList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	globalShort := fs.Bool("g", false, "show global/user scope")
	globalLong := fs.Bool("global", false, "show global/user scope")
	localShort := fs.Bool("l", false, "show local/project scope")
	localLong := fs.Bool("local", false, "show local/project scope")
	projectLong := fs.Bool("project", false, "show local/project scope")
	availableShort := fs.Bool("a", false, "list available skills from registry")
	availableLong := fs.Bool("available", false, "list available skills from registry")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode,cursor,amp,kilocode,roo,goose,antigravity,copilot,clawdbot,droid,windsurf")
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

	if *availableShort || *availableLong {
		return a.runListAvailable(skillFilter)
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
	items, err := skill.List(scopes, cwd, tools)
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
		if !matchesSkillFilter(item.Name, skillFilter) {
			continue
		}
		if writer == nil || item.Client != lastTool || item.Scope != lastScope {
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
			fmt.Fprintln(writer, "SKILL\tVERSION\tDESCRIPTION\tPATH")
			printed = true
			lastTool = item.Client
			lastScope = item.Scope
		}
		matched++
		version, _ := readSkillVersion(item.Path)
		description := ""
		meta, err := loadSkillMeta(item.Path)
		if err == nil {
			description = meta.Description
		} else {
			description, _ = readSkillDescription(item.Path)
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", item.Name, version, description, item.Path)
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

func (a *App) runListAvailable(filter string) int {
	err := cli.RunWithSpinner(a.errOut, "", cli.DefaultTips(), time.Second, func() error {
		return registryindex.EnsureIndexes()
	})
	if err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}
	index, err := registryindex.LoadSkillIndex()
	if err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "SKILL\tUPDATED\tDESCRIPTION")
	matched := 0
	for _, entry := range index.Skills {
		if filter != "" && !matchesSkillFilter(entry.Name, filter) {
			continue
		}
		meta, err := fetchRemoteSkillMeta(entry.Name)
		if err != nil && !isRemoteNotFound(err) {
			fmt.Fprintf(a.errOut, "list failed: %v\n", err)
			return 1
		}
		updatedAt := entry.UpdatedAt
		description := ""
		if err == nil {
			if meta.UpdatedAt != "" {
				updatedAt = meta.UpdatedAt
			}
			description = meta.Description
		}
		description = truncateDescription(description, 80)
		fmt.Fprintf(writer, "%s\t%s\t%s\n", entry.Name, updatedAt, description)
		matched++
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(a.errOut, "list failed: %v\n", err)
		return 1
	}
	if matched == 0 {
		fmt.Fprintln(a.out, "no matching skills found")
	}
	return 0
}

func (a *App) printListHelp() {
	fmt.Fprintf(a.out, `Usage: %s list [skill] [--global|-g] [--local|-l] [--client|-c <list>] [--available|-a]

Examples:
  %s list
  %s list --available
  %s list my-skill -l
  %s list my-skill -g -c opencode
  %s list -g
`, a.binaryName, a.binaryName, a.binaryName, a.binaryName, a.binaryName)
}
