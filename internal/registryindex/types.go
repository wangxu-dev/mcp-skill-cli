package registryindex

type SkillEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Repo        string `json:"repo"`
	Head        string `json:"head"`
	UpdatedAt   string `json:"updatedAt"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

type SkillIndex struct {
	GeneratedAt string       `json:"generatedAt"`
	Skills      []SkillEntry `json:"skills"`
}

type MCPEntry struct {
	Name        string            `json:"name"`
	Type        string            `json:"type,omitempty"`
	Description string            `json:"description,omitempty"`
	Path        string            `json:"path,omitempty"`
	Repo        string            `json:"repo,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Requires    []string          `json:"requires,omitempty"`
	Install     []string          `json:"install,omitempty"`
	Run         MCPRun            `json:"run,omitempty"`
	Inputs      []MCPInput        `json:"inputs,omitempty"`
	Head        string            `json:"head,omitempty"`
	UpdatedAt   string            `json:"updatedAt,omitempty"`
	CheckedAt   string            `json:"checkedAt,omitempty"`
}

type MCPRun struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type MCPInput struct {
	Name     string   `json:"name"`
	Label    string   `json:"label,omitempty"`
	Type     string   `json:"type,omitempty"`
	Required bool     `json:"required,omitempty"`
	Default  string   `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"`
}

type MCPIndex struct {
	GeneratedAt string     `json:"generatedAt"`
	MCP         []MCPEntry `json:"mcp"`
	Servers     []MCPEntry `json:"servers"`
}
