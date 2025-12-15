# MCP get_skill Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `get_skill` tool to the Bytebase MCP server that provides step-by-step guides for common tasks, making the MCP self-documenting.

**Architecture:** Embedded markdown skill files parsed at startup. Tool description lists skill names and workflow; calling the tool returns full skill content. Skills follow Claude skill format with YAML frontmatter.

**Tech Stack:** Go, `//go:embed`, YAML frontmatter parsing, MCP SDK

---

## Task 1: Create skills directory and execute-sql.md

**Files:**
- Create: `backend/api/mcp/skills/execute-sql.md`

**Step 1: Create skills directory**

```bash
mkdir -p backend/api/mcp/skills
```

**Step 2: Create execute-sql.md**

```markdown
---
name: execute-sql
description: Run SQL queries on databases
---

# Execute SQL

## Overview

Run SQL queries against databases managed by Bytebase.

## Prerequisites

- Know the instance and database name
- Have `bb.databases.query` permission

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="SQLService/Query")
   ```

2. **List databases in project** (if needed):
   ```
   call_api(operationId="DatabaseService/ListDatabases", body={
     "parent": "projects/{project-id}"
   })
   ```

3. **Execute SQL**:
   ```
   call_api(operationId="SQLService/Query", body={
     "name": "instances/{instance-id}/databases/{database-name}",
     "statement": "SELECT * FROM users LIMIT 10"
   })
   ```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong instance/database name | List databases first |
| permission denied | Missing bb.databases.query | Check user permissions |
| syntax error | Invalid SQL | Check SQL syntax for the database engine |
```

**Step 3: Verify file created**

```bash
cat backend/api/mcp/skills/execute-sql.md
```

---

## Task 2: Create create-instance.md

**Files:**
- Create: `backend/api/mcp/skills/create-instance.md`

**Step 1: Create create-instance.md**

```markdown
---
name: create-instance
description: Add a database instance to Bytebase
---

# Create Instance

## Overview

Add a database instance (PostgreSQL, MySQL, etc.) to Bytebase for management.

## Prerequisites

- Have `bb.instances.create` permission
- Know the database engine type and connection details

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="InstanceService/CreateInstance")
   ```

2. **List environments** (to get environment resource name):
   ```
   call_api(operationId="EnvironmentService/ListEnvironments", body={})
   ```

3. **Create the instance**:
   ```
   call_api(operationId="InstanceService/CreateInstance", body={
     "parent": "projects/{project-id}",
     "instance": {
       "title": "Production PostgreSQL",
       "engine": "POSTGRES",
       "environment": "environments/prod",
       "activation": true,
       "dataSources": [{
         "type": "ADMIN",
         "host": "localhost",
         "port": "5432",
         "username": "admin",
         "password": "secret"
       }]
     },
     "instanceId": "prod-pg"
   })
   ```

## Engine Types

POSTGRES, MYSQL, TIDB, CLICKHOUSE, SNOWFLAKE, SQLITE, MONGODB, REDIS, ORACLE, MSSQL, MARIADB, OCEANBASE

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| environment not found | Invalid environment name | List environments first |
| instance already exists | Duplicate instanceId | Choose different instanceId |
| connection failed | Wrong host/port/credentials | Verify connection details |
```

---

## Task 3: Create create-project.md

**Files:**
- Create: `backend/api/mcp/skills/create-project.md`

**Step 1: Create create-project.md**

```markdown
---
name: create-project
description: Set up a new Bytebase project
---

# Create Project

## Overview

Create a new project to organize databases and team members.

## Prerequisites

- Have `bb.projects.create` permission

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="ProjectService/CreateProject")
   ```

2. **Create the project**:
   ```
   call_api(operationId="ProjectService/CreateProject", body={
     "project": {
       "title": "My Project",
       "key": "MYPROJ"
     },
     "projectId": "my-project"
   })
   ```

3. **Add team members** (optional):
   ```
   search_api(operationId="ProjectService/SetProjectIamPolicy")
   ```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| project already exists | Duplicate projectId | Choose different projectId |
| invalid key | Key format wrong | Use uppercase letters, 2-10 chars |
```

---

## Task 4: Create schema-change.md

**Files:**
- Create: `backend/api/mcp/skills/schema-change.md`

**Step 1: Create schema-change.md**

```markdown
---
name: schema-change
description: Create schema migration issues
---

# Schema Change

## Overview

Create a schema change issue to modify database structure (DDL) through Bytebase's review workflow.

## Prerequisites

- Have `bb.issues.create` permission
- Know the target database

## Workflow

1. **Get the schema**:
   ```
   search_api(operationId="IssueService/CreateIssue")
   ```

2. **Get database info**:
   ```
   call_api(operationId="DatabaseService/GetDatabase", body={
     "name": "instances/{instance-id}/databases/{database-name}"
   })
   ```

3. **Create schema change issue**:
   ```
   call_api(operationId="IssueService/CreateIssue", body={
     "parent": "projects/{project-id}",
     "issue": {
       "title": "Add users table",
       "type": "DATABASE_CHANGE",
       "plan": {
         "steps": [{
           "specs": [{
             "changeDatabaseConfig": {
               "target": "instances/{instance}/databases/{database}",
               "type": "MIGRATE",
               "sheet": "projects/{project}/sheets/{sheet-id}"
             }
           }]
         }]
       }
     }
   })
   ```

## Sheet Creation

Before creating an issue, you may need to create a sheet with the SQL:

```
call_api(operationId="SheetService/CreateSheet", body={
  "parent": "projects/{project-id}",
  "sheet": {
    "title": "Add users table",
    "content": "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));"
  }
})
```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| database not found | Wrong database reference | Verify database resource name |
| sheet not found | Sheet doesn't exist | Create sheet first |
```

---

## Task 5: Create tool_skill_test.go

**Files:**
- Create: `backend/api/mcp/tool_skill_test.go`

**Step 1: Create the test file**

```go
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
			require.NotContains(t, text, "not found", "skill %s should be loadable", skill)
			require.NotEmpty(t, text)
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/mcp -run ^TestGetSkill
```

Expected: FAIL - `handleGetSkill` not defined

---

## Task 6: Create tool_skill.go

**Files:**
- Create: `backend/api/mcp/tool_skill.go`

**Step 1: Create the implementation file**

```go
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

Skills: execute-sql, create-instance, create-project, schema-change`

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
	text, err := s.getSkillContent(input.Skill)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, nil, nil
	}

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

func (*Server) getSkillContent(skillName string) (string, error) {
	filename := skillName + ".md"
	content, err := skillFiles.ReadFile(filepath.Join("skills", filename))
	if err != nil {
		return "", fmt.Errorf("skill %q not found. Use get_skill() to list available skills", skillName)
	}

	// Strip frontmatter for cleaner output
	str := string(content)
	if strings.HasPrefix(str, "---") {
		if end := strings.Index(str[3:], "---"); end != -1 {
			str = strings.TrimSpace(str[3+end+3:])
		}
	}

	return str, nil
}
```

**Step 2: Verify it compiles**

```bash
go build ./backend/api/mcp/...
```

---

## Task 7: Register skill tool in server.go

**Files:**
- Modify: `backend/api/mcp/server.go:59-62`

**Step 1: Add registerSkillTool to registerTools**

Change:

```go
func (s *Server) registerTools() {
	s.registerSearchTool()
	s.registerCallTool()
}
```

To:

```go
func (s *Server) registerTools() {
	s.registerSearchTool()
	s.registerCallTool()
	s.registerSkillTool()
}
```

**Step 2: Run tests**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/mcp -run ^TestGetSkill
```

Expected: All PASS

---

## Task 8: Run all MCP tests

**Step 1: Run full test suite**

```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/mcp/...
```

Expected: All PASS

---

## Task 9: Lint and tidy

**Step 1: Tidy go modules**

```bash
go mod tidy
```

**Step 2: Run linter**

```bash
golangci-lint run --allow-parallel-runners ./backend/api/mcp/...
```

**Step 3: Fix any issues and re-run until clean**

---

## Task 10: Build and verify

**Step 1: Build the project**

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Build succeeds

---

## Task 11: Commit

**Step 1: Stage files**

```bash
git add backend/api/mcp/skills/ backend/api/mcp/tool_skill.go backend/api/mcp/tool_skill_test.go backend/api/mcp/server.go
```

**Step 2: Commit**

```bash
git commit -m "feat(mcp): add get_skill tool for self-documenting guides

Add a get_skill tool that provides step-by-step guides for common
Bytebase tasks. Skills are embedded markdown files following the
Claude skill format with YAML frontmatter.

Initial skills:
- execute-sql: Run SQL queries
- create-instance: Add database instance
- create-project: Set up project
- schema-change: Create schema migrations"
```
