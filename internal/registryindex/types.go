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
	Name      string `json:"name"`
	Path      string `json:"path"`
	Repo      string `json:"repo"`
	Head      string `json:"head"`
	UpdatedAt string `json:"updatedAt"`
}

type MCPIndex struct {
	GeneratedAt string     `json:"generatedAt"`
	MCP         []MCPEntry `json:"mcp"`
	Servers     []MCPEntry `json:"servers"`
}
