package mcpcli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"mcp-skill-manager/internal/installer"
	"mcp-skill-manager/internal/mcp"
	"mcp-skill-manager/internal/registryindex"
)

type registryInstallOptions struct {
	Scope   string
	Cwd     string
	Clients []installer.Tool
	Force   bool
	Out     *bufio.Writer
	ErrOut  *bufio.Writer
}

func installFromRegistryEntry(entry registryindex.MCPEntry, opts registryInstallOptions) ([]mcp.Installed, error) {
	entryType := normalizeEntryType(entry)
	if entryType == "" {
		return nil, fmt.Errorf("invalid mcp entry: missing type")
	}

	requirements := normalizeRequirements(entry.Requires, entryType)
	if err := checkRequirements(requirements); err != nil {
		return nil, err
	}

	inputs, err := collectInputs(entry.Inputs, opts.Out)
	if err != nil {
		return nil, err
	}

	var repoPath string
	if entryType == "stdio" {
		if strings.TrimSpace(entry.Repo) == "" {
			return nil, fmt.Errorf("invalid mcp entry: missing repo")
		}
		repoPath, repoUpdated, err := prepareRepo(entry, opts.Force, opts.Out)
		if err != nil {
			return nil, err
		}
		if repoUpdated || opts.Force {
			if err := runInstallSteps(entry.Install, repoPath); err != nil {
				return nil, err
			}
		}
	}

	def, err := buildDefinitionFromEntry(entry, inputs, repoPath)
	if err != nil {
		return nil, err
	}

	templateDef, err := buildTemplateDefinition(entry, repoPath)
	if err != nil {
		return nil, err
	}
	if _, err := mcp.SaveLocalDefinition(templateDef); err != nil {
		return nil, err
	}

	records, err := mcp.Install(def, opts.Scope, opts.Cwd, opts.Clients, opts.Force)
	if err != nil {
		return nil, err
	}

	if err := registryindex.SaveLocalRecord("mcp", registryindex.LocalRecord{
		Name:      entry.Name,
		Repo:      entry.Repo,
		Path:      entry.Path,
		Head:      entry.Head,
		UpdatedAt: entry.UpdatedAt,
	}); err != nil {
		return nil, err
	}

	return records, nil
}

func normalizeEntryType(entry registryindex.MCPEntry) string {
	entryType := strings.ToLower(strings.TrimSpace(entry.Type))
	if entryType == "" && strings.TrimSpace(entry.URL) != "" {
		entryType = "http"
	}
	if entryType == "" && strings.TrimSpace(entry.Repo) != "" {
		entryType = "stdio"
	}
	return entryType
}

func normalizeRequirements(requirements []string, entryType string) []string {
	seen := map[string]bool{}
	var result []string
	for _, req := range requirements {
		req = strings.ToLower(strings.TrimSpace(req))
		if req == "" || seen[req] {
			continue
		}
		seen[req] = true
		result = append(result, req)
	}
	if entryType == "stdio" && !seen["git"] {
		result = append(result, "git")
	}
	return result
}

func checkRequirements(requirements []string) error {
	var missing []string
	for _, req := range requirements {
		if req == "" {
			continue
		}
		if _, err := exec.LookPath(req); err != nil {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing requirements: %s", strings.Join(missing, ", "))
	}
	return nil
}

func collectInputs(inputs []registryindex.MCPInput, out *bufio.Writer) (map[string]string, error) {
	values := make(map[string]string, len(inputs))
	reader := bufio.NewReader(os.Stdin)
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return nil, fmt.Errorf("invalid input: missing name")
		}
		label := strings.TrimSpace(input.Label)
		if label == "" {
			label = name
		}
		value, err := promptInput(reader, out, label, input)
		if err != nil {
			return nil, err
		}
		values[name] = value
	}
	return values, nil
}

func promptInput(reader *bufio.Reader, out *bufio.Writer, label string, input registryindex.MCPInput) (string, error) {
	for {
		prompt := label
		if input.Type == "choice" && len(input.Options) > 0 {
			prompt = fmt.Sprintf("%s (%s)", label, strings.Join(input.Options, "/"))
		}
		if input.Default != "" {
			prompt = fmt.Sprintf("%s [%s]", prompt, input.Default)
		}
		if input.Type == "bool" {
			prompt = fmt.Sprintf("%s (y/n)", prompt)
		}
		fmt.Fprintf(out, "%s: ", prompt)
		if err := out.Flush(); err != nil {
			return "", err
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = input.Default
		}
		if value == "" && input.Required {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(input.Type)) {
		case "choice":
			if len(input.Options) == 0 {
				return value, nil
			}
			if idx, err := strconv.Atoi(value); err == nil && idx >= 1 && idx <= len(input.Options) {
				return input.Options[idx-1], nil
			}
			if isOptionMatch(input.Options, value) {
				return value, nil
			}
			fmt.Fprintln(out, "Invalid choice. Try again.")
			_ = out.Flush()
			continue
		case "bool":
			if value == "" && !input.Required {
				return "", nil
			}
			switch strings.ToLower(value) {
			case "y", "yes", "true", "1":
				return "true", nil
			case "n", "no", "false", "0":
				return "false", nil
			}
			fmt.Fprintln(out, "Invalid choice. Use y/n.")
			_ = out.Flush()
			continue
		default:
			return value, nil
		}
	}
}

func isOptionMatch(options []string, value string) bool {
	for _, opt := range options {
		if strings.EqualFold(opt, value) {
			return true
		}
	}
	return false
}

func prepareRepo(entry registryindex.MCPEntry, force bool, out *bufio.Writer) (string, bool, error) {
	root, err := installer.LocalMcpStore()
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", false, err
	}
	dest := filepath.Join(root, entry.Name)
	if _, err := os.Stat(dest); err == nil {
		needs, err := needsMcpUpdate(entry)
		if err != nil {
			return "", false, err
		}
		if !needs && !force {
			return dest, false, nil
		}
		if !force {
			if !confirmUpdate(out, entry.Name) {
				return dest, false, nil
			}
		}
		if err := os.RemoveAll(dest); err != nil {
			return "", false, err
		}
	}

	repo, err := normalizeRepoURL(entry.Repo)
	if err != nil {
		return "", false, err
	}
	if err := gitClone(repo, dest); err != nil {
		return "", false, err
	}
	if strings.TrimSpace(entry.Head) != "" {
		if err := gitCheckout(dest, entry.Head); err != nil {
			return "", false, err
		}
	}
	return dest, true, nil
}

func normalizeRepoURL(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", fmt.Errorf("repository is required")
	}
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo, nil
	}
	if strings.Count(repo, "/") == 1 {
		return "https://github.com/" + repo + ".git", nil
	}
	return "", fmt.Errorf("unsupported repository format: %s", repo)
}

func gitClone(repoURL, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, dest)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git clone failed: %s", msg)
	}
	return nil
}

func gitCheckout(repoPath, head string) error {
	cmd := exec.Command("git", "fetch", "--depth", "1", "origin", head)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %v", err)
	}
	cmd = exec.Command("git", "checkout", head)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %v", err)
	}
	return nil
}

func needsMcpUpdate(entry registryindex.MCPEntry) (bool, error) {
	record, ok, err := registryindex.LocalRecordFor("mcp", entry.Name)
	if err != nil {
		return true, err
	}
	if !ok {
		return true, nil
	}
	if entry.Head != "" && record.Head != "" && entry.Head == record.Head {
		return false, nil
	}
	if entry.UpdatedAt != "" && record.UpdatedAt != "" {
		entryTime, err := time.Parse(time.RFC3339, entry.UpdatedAt)
		if err != nil {
			return true, nil
		}
		recordTime, err := time.Parse(time.RFC3339, record.UpdatedAt)
		if err != nil {
			return true, nil
		}
		if !entryTime.After(recordTime) {
			return false, nil
		}
	}
	if entry.Head != "" && record.Head == entry.Head {
		return false, nil
	}
	return true, nil
}

func confirmUpdate(out *bufio.Writer, name string) bool {
	if out == nil {
		return true
	}
	fmt.Fprintf(out, "Cached MCP '%s' exists. Update cache? Type 'yes' to continue: ", name)
	_ = out.Flush()
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "yes"
}

func runInstallSteps(steps []string, repoPath string) error {
	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}
		if err := runShellCommand(step, repoPath); err != nil {
			return err
		}
	}
	return nil
}

func runShellCommand(command, repoPath string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = repoPath
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("install step failed (%s): %s", command, msg)
	}
	return nil
}

func buildDefinitionFromEntry(entry registryindex.MCPEntry, inputs map[string]string, repoPath string) (mcp.Definition, error) {
	entryType := normalizeEntryType(entry)
	if repoPath != "" {
		inputs = withBuiltins(inputs, map[string]string{
			"ROOT": repoPath,
		})
	}
	switch entryType {
	case "http":
		url := expandPlaceholders(entry.URL, inputs)
		def, err := mcp.DefinitionFromArgs(entry.Name, "http", url, "", nil)
		if err != nil {
			return mcp.Definition{}, err
		}
		def.Headers = expandMap(entry.Headers, inputs)
		return def, nil
	case "stdio":
		command := expandPlaceholders(entry.Run.Command, inputs)
		args := expandSlice(entry.Run.Args, inputs)
		env := expandMap(entry.Run.Env, inputs)
		def, err := mcp.DefinitionFromArgs(entry.Name, "stdio", "", command, args)
		if err != nil {
			return mcp.Definition{}, err
		}
		def.Env = env
		return def, nil
	default:
		return mcp.Definition{}, fmt.Errorf("unsupported transport: %s", entryType)
	}
}

func buildTemplateDefinition(entry registryindex.MCPEntry, repoPath string) (mcp.Definition, error) {
	inputs := map[string]string{}
	if repoPath != "" {
		inputs["ROOT"] = repoPath
	}
	entryType := normalizeEntryType(entry)
	switch entryType {
	case "http":
		def, err := mcp.DefinitionFromArgs(entry.Name, "http", expandPlaceholders(entry.URL, inputs), "", nil)
		if err != nil {
			return mcp.Definition{}, err
		}
		def.Headers = expandMap(entry.Headers, inputs)
		return def, nil
	case "stdio":
		command := expandPlaceholders(entry.Run.Command, inputs)
		args := expandSlice(entry.Run.Args, inputs)
		env := expandMap(entry.Run.Env, inputs)
		def, err := mcp.DefinitionFromArgs(entry.Name, "stdio", "", command, args)
		if err != nil {
			return mcp.Definition{}, err
		}
		def.Env = env
		return def, nil
	default:
		return mcp.Definition{}, fmt.Errorf("unsupported transport: %s", entryType)
	}
}

func withBuiltins(inputs map[string]string, builtins map[string]string) map[string]string {
	if len(builtins) == 0 {
		return inputs
	}
	merged := make(map[string]string, len(inputs)+len(builtins))
	for key, value := range inputs {
		merged[key] = value
	}
	for key, value := range builtins {
		if _, exists := merged[key]; !exists {
			merged[key] = value
		}
	}
	return merged
}

func expandPlaceholders(value string, inputs map[string]string) string {
	if value == "" || len(inputs) == 0 {
		return value
	}
	for key, val := range inputs {
		value = strings.ReplaceAll(value, "${"+key+"}", val)
	}
	return value
}

func expandSlice(values []string, inputs map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	expanded := make([]string, 0, len(values))
	for _, value := range values {
		expanded = append(expanded, expandPlaceholders(value, inputs))
	}
	return expanded
}

func expandMap(values map[string]string, inputs map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	expanded := make(map[string]string, len(values))
	for key, value := range values {
		expanded[key] = expandPlaceholders(value, inputs)
	}
	return expanded
}
