package mcp

import (
	"context"
	"regexp"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestGetSkillListSkills(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test listing all skills (no parameters)
	result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Available Skills")
	require.Contains(t, text, "query")
	require.Contains(t, text, "database-change")
}

func TestGetSkillSpecificSkill(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test getting query skill
	result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{
		Skill: "query",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Execute SQL")
	require.Contains(t, text, "SQLService/Query")
	require.Contains(t, text, "Workflow")
}

func TestGetSkillNotFound(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	// Test getting non-existent skill
	result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{
		Skill: "non-existent",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "not found")
	require.Contains(t, text, "get_skill()")
}

func TestGetSkillAllSkillsLoadable(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := newServerWithStore(newTestServerStore(), profile, "test-secret", nil)
	require.NoError(t, err)

	skills := []string{"query", "database-change"}
	for _, skill := range skills {
		t.Run(skill, func(t *testing.T) {
			result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{
				Skill: skill,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1)

			text := result.Content[0].(*mcpsdk.TextContent).Text
			require.NotContains(t, text, "skill \""+skill+"\" not found", "skill %s should be loadable", skill)
			require.NotEmpty(t, text)
		})
	}
}

// TestSkillsOnlyInstructServableCalls is the rule that keeps a shipped skill
// honest: every operation a skill tells the agent to call must be one the MCP
// ceiling gate serves. It states the rule rather than fixing an instance of it,
// because the failure mode is silent — a classification change lands in a proto
// file, the skill keeps shipping, and the agent follows a playbook whose every
// step comes back 403.
//
// That is exactly what happened to the grant-permission skill: 1b-1 classified
// the five IAM operations it drove as EXCLUDED, 1b-2 turned EXCLUDED into a
// denial, and nothing connected the two. The skill is gone; this is what stops
// the next one going the same way.
//
// It resolves through the same index call_api uses, so an operation ID that no
// longer resolves fails here too.
func TestSkillsOnlyInstructServableCalls(t *testing.T) {
	idx, err := NewOpenAPIIndex()
	require.NoError(t, err)

	entries, err := skillFiles.ReadDir("skills")
	require.NoError(t, err)
	require.NotEmpty(t, entries, "the embedded skills directory must not be empty")

	operationID := regexp.MustCompile(`operationId="([^"]+)"`)
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			content, err := skillFiles.ReadFile("skills/" + entry.Name())
			require.NoError(t, err)

			matches := operationID.FindAllStringSubmatch(string(content), -1)
			require.NotEmpty(t, matches, "a skill that instructs no call is not a skill")
			for _, match := range matches {
				operation := match[1]
				endpoint, ok := idx.GetEndpoint(operation)
				require.True(t, ok, "%s instructs operation %q, which call_api cannot resolve", entry.Name(), operation)
				class, err := auth.MCPMethodClassOfProcedure(endpoint.Path)
				require.NoError(t, err, "%s instructs %q, whose classification cannot be read", entry.Name(), operation)
				require.False(t, auth.MCPClassIsRefused(class),
					"%s instructs %q, which is %v — no MCP ceiling serves it, so the skill cannot be followed",
					entry.Name(), operation, class)
			}
		})
	}
}
