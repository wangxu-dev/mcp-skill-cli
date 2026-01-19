package skillcli

import (
	"flag"
	"fmt"
	"os"

	"mcp-skill-manager/internal/cli"
	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/registryindex"
	"mcp-skill-manager/internal/skill"
)

func (a *App) runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(a.errOut)
	globalShort := fs.Bool("g", false, "update global/user scope")
	globalLong := fs.Bool("global", false, "update global/user scope")
	localShort := fs.Bool("l", false, "update local/project scope")
	localLong := fs.Bool("local", false, "update local/project scope")
	projectLong := fs.Bool("project", false, "update local/project scope")
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
		a.printUpdateHelp()
		return 0
	}
	if len(positionals) > 1 {
		fmt.Fprintln(a.errOut, "update accepts at most one skill name")
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
	tools, err := installer.ParseTools(clientValue)
	if err != nil {
		fmt.Fprintf(a.errOut, "invalid client list: %v\n", err)
		return 2
	}
	scopes := resolveListScopes(*globalShort || *globalLong, *localShort || *localLong || *projectLong)
	cwd, _ := os.Getwd()
	items, err := skill.List(scopes, cwd, tools)
	if err != nil {
		fmt.Fprintf(a.errOut, "update failed: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(a.out, "no skills installed")
		return 0
	}

	var targets []skill.Installed
	for _, item := range items {
		if nameFilter != "" && item.Name != nameFilter {
			continue
		}
		targets = append(targets, item)
	}
	if len(targets) == 0 {
		fmt.Fprintln(a.out, "no matching skills found")
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
		item    skill.Installed
		message string
		err     error
	}
	var results []result
	for _, item := range targets {
		entry, ok, err := registryindex.FindSkill(item.Name)
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		if !ok {
			results = append(results, result{item: item, message: "not in registry"})
			continue
		}
		remoteMeta, remoteErr := fetchRemoteSkillMeta(entry.Name)
		if remoteErr != nil {
			localPath, err := localStoreSkillPath(item.Name)
			if err != nil {
				results = append(results, result{item: item, err: err})
				continue
			}
			if err := registryindex.SyncSkill(entry); err != nil {
				results = append(results, result{item: item, err: err})
				continue
			}
			needsUpdate, installedVersion, cachedVersion, err := needsSkillUpdate(item.Path, localPath)
			if err != nil {
				results = append(results, result{item: item, err: err})
				continue
			}
			if !needsUpdate {
				label := "already latest"
				if installedVersion != "" {
					label = fmt.Sprintf("already latest (%s)", installedVersion)
				}
				results = append(results, result{item: item, message: label})
				continue
			}
			records, err := skill.Install(entry.Name, item.Scope, cwd, []installer.Tool{item.Client}, true)
			if err != nil {
				results = append(results, result{item: item, err: err})
				continue
			}
			_ = records
			msg := "updated"
			if cachedVersion != "" {
				msg = fmt.Sprintf("updated to %s", cachedVersion)
			}
			results = append(results, result{item: item, message: msg})
			continue
		}

		installedMeta, _ := loadSkillMeta(item.Path)
		installedVersion, _ := readSkillVersion(item.Path)
		if installedVersion == "" {
			installedVersion = installedMeta.Version
		}
		if remoteMeta.Version != "" && installedVersion != "" && remoteMeta.Version == installedVersion {
			label := fmt.Sprintf("already latest (%s)", installedVersion)
			results = append(results, result{item: item, message: label})
			continue
		}
		if remoteMeta.Head != "" && installedMeta.Head != "" && remoteMeta.Head == installedMeta.Head {
			label := "already latest"
			if installedVersion != "" {
				label = fmt.Sprintf("already latest (%s)", installedVersion)
			}
			results = append(results, result{item: item, message: label})
			continue
		}

		if err := registryindex.SyncSkill(entry); err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		records, err := skill.Install(entry.Name, item.Scope, cwd, []installer.Tool{item.Client}, true)
		if err != nil {
			results = append(results, result{item: item, err: err})
			continue
		}
		_ = records
		msg := "updated"
		if remoteMeta.Version != "" {
			msg = fmt.Sprintf("updated to %s", remoteMeta.Version)
		}
		results = append(results, result{item: item, message: msg})
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
  %s update work-session -l -c claude
`, a.binaryName, a.binaryName, a.binaryName)
}
