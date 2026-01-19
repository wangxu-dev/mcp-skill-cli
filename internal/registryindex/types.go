package registryindex

type SkillEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Repo      string `json:"repo"`
	Head      string `json:"head"`
	UpdatedAt string `json:"updatedAt"`
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
	Head        string            `json:"head,omitempty"`
	UpdatedAt   string            `json:"updatedAt,omitempty"`
	CheckedAt   string            `json:"checkedAt,omitempty"`
}

type MCPIndex struct {
	GeneratedAt string     `json:"generatedAt"`
	MCP         []MCPEntry `json:"mcp"`
	Servers     []MCPEntry `json:"servers"`
}
