package mcp

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

//go:embed skills/*.md
var skillFiles embed.FS

// SkillInput is the input for the get_skill tool.
type SkillInput struct {
	// Skill is the name of the skill to retrieve.
	// Leave empty to list all available skills.
	Skill string `json:"skill,omitempty"`
}

// skillMeta holds parsed skill metadata from frontmatter.
type skillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// getSkillDescription is the description for the get_skill tool.
const getSkillDescription = `Get step-by-step guides for Bytebase tasks.

**Workflow:** get_skill("task") → search_api(operationId) → call_api(...)

Skills: query, database-change, grant-permission`

func (s *Server) registerSkillTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_skill",
		Description: getSkillDescription,
	}, s.handleGetSkill)
}

func (s *Server) handleGetSkill(_ context.Context, _ *mcp.CallToolRequest, input SkillInput) (*mcp.CallToolResult, any, error) {
	if input.Skill == "" {
		// List all skills
		text := s.formatSkillList()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}

	// Get specific skill
	text := s.getSkillContent(input.Skill)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func (*Server) formatSkillList() string {
	var sb strings.Builder
	sb.WriteString("## Available Skills\n\n")
	sb.WriteString("Use `get_skill(skill=\"name\")` to get the full guide.\n\n")
	sb.WriteString("| Skill | Description |\n")
	sb.WriteString("|-------|-------------|\n")

	entries, _ := skillFiles.ReadDir("skills")
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			meta := parseSkillMeta(entry.Name())
			if meta != nil {
				sb.WriteString(fmt.Sprintf("| %s | %s |\n", meta.Name, meta.Description))
			}
		}
	}

	return sb.String()
}

func parseSkillMeta(filename string) *skillMeta {
	content, err := skillFiles.ReadFile(filepath.Join("skills", filename))
	if err != nil {
		return nil
	}

	// Parse YAML frontmatter
	str := string(content)
	if !strings.HasPrefix(str, "---") {
		return nil
	}

	end := strings.Index(str[3:], "---")
	if end == -1 {
		return nil
	}

	var meta skillMeta
	if err := yaml.Unmarshal([]byte(str[3:3+end]), &meta); err != nil {
		return nil
	}

	return &meta
}

func (*Server) getSkillContent(skillName string) string {
	filename := skillName + ".md"
	content, err := skillFiles.ReadFile(filepath.Join("skills", filename))
	if err != nil {
		return fmt.Sprintf("Skill %q not found. Use get_skill() to list available skills.", skillName)
	}

	// Strip frontmatter for cleaner output
	str := string(content)
	if strings.HasPrefix(str, "---") {
		if end := strings.Index(str[3:], "---"); end != -1 {
			str = strings.TrimSpace(str[3+end+3:])
		}
	}

	return str
}
