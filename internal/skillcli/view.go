package skillcli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/registryindex"
	"mcp-skill-manager/internal/skill"
)

func (a *App) runView(args []string) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	installedFlag := fs.Bool("installed", false, "show installed skill info")
	globalShort := fs.Bool("g", false, "show global/user scope (installed only)")
	globalLong := fs.Bool("global", false, "show global/user scope (installed only)")
	localShort := fs.Bool("l", false, "show local/project scope (installed only)")
	localLong := fs.Bool("local", false, "show local/project scope (installed only)")
	projectLong := fs.Bool("project", false, "show local/project scope (installed only)")
	clientFlag := fs.String("client", "", "comma-separated clients: claude,codex,gemini,opencode,cursor,amp,kilocode,roo,goose,antigravity,copilot,clawdbot,droid,windsurf (installed only)")
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
		fmt.Fprintln(a.errOut, "view requires a skill name")
		return 2
	}
	if len(positionals) > 1 {
		fmt.Fprintln(a.errOut, "view accepts a single skill name")
		return 2
	}

	name := positionals[0]
	if *installedFlag {
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
			fmt.Fprintf(a.errOut, "view failed: %v\n", err)
			return 1
		}

		var matches []skill.Installed
		for _, item := range items {
			if item.Name == name {
				matches = append(matches, item)
			}
		}
		if len(matches) == 0 {
			fmt.Fprintln(a.out, "no matching skills found")
			return 0
		}
		for idx, item := range matches {
			if idx > 0 {
				fmt.Fprintln(a.out)
			}
			fmt.Fprintf(a.out, "%s (%s)\n", item.Client, item.Scope)
			meta, err := loadSkillMeta(item.Path)
			if err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(a.errOut, "view failed: %v\n", err)
				return 1
			}
			version, _ := readSkillVersion(item.Path)
			printSkillMeta(a.out, item.Name, meta, version)
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
	entry, ok, err := registryindex.FindSkill(name)
	if err != nil {
		fmt.Fprintf(a.errOut, "view failed: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(a.out, "skill not found in registry")
		return 0
	}
	meta, err := fetchRemoteSkillMeta(entry.Name)
	if err != nil {
		if isRemoteNotFound(err) {
			fmt.Fprintln(a.out, "skill not found in registry")
			return 0
		}
		fmt.Fprintf(a.errOut, "view failed: %v\n", err)
		return 1
	}
	printSkillMeta(a.out, entry.Name, meta, meta.Version)
	return 0
}

func (a *App) printViewHelp() {
	fmt.Fprintf(a.out, `Usage: %s view <name> [--installed] [--global|-g] [--local|-l] [--client|-c <list>]

Examples:
  %s view work-session
  %s view work-session --installed -g -c claude
`, a.binaryName, a.binaryName, a.binaryName)
}
