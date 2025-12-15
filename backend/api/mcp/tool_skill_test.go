package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

func TestGetSkillListSkills(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test listing all skills (no parameters)
	result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	text := result.Content[0].(*mcpsdk.TextContent).Text
	require.Contains(t, text, "Available Skills")
	require.Contains(t, text, "execute-sql")
	require.Contains(t, text, "create-instance")
}

func TestGetSkillSpecificSkill(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev}
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	// Test getting execute-sql skill
	result, _, err := s.handleGetSkill(context.Background(), nil, SkillInput{
		Skill: "execute-sql",
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
	s, err := NewServer(nil, profile, "test-secret")
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
	s, err := NewServer(nil, profile, "test-secret")
	require.NoError(t, err)

	skills := []string{"execute-sql", "create-instance", "create-project", "schema-change"}
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
