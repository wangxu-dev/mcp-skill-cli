package skillcli

import (
	"fmt"
	"io"
)

func printSkillMeta(out io.Writer, name string, meta SkillMeta, version string) {
	displayName := name
	if meta.Name != "" {
		displayName = meta.Name
	}
	fmt.Fprintf(out, "name: %s\n", displayName)
	if version != "" {
		fmt.Fprintf(out, "version: %s\n", version)
	}
	if meta.Description != "" {
		fmt.Fprintf(out, "description: %s\n", meta.Description)
	}
	if meta.UpdatedAt != "" {
		fmt.Fprintf(out, "updatedAt: %s\n", meta.UpdatedAt)
	}
}
