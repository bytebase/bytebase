# MCP Server Skills

This directory contains the Bytebase MCP server implementation with embedded skills that provide step-by-step guides for common tasks.

## Writing Skills

Skills are markdown files in `skills/` that get embedded into the binary and served via the `get_skill` tool.

### File Structure

```
skills/
├── execute-sql.md
├── create-instance.md
├── create-project.md
└── schema-change.md
```

### Skill Format

Each skill file must have YAML frontmatter followed by markdown content:

```markdown
---
name: skill-name
description: Short description for the skill list
---

# Skill Title

## Overview

One-liner explaining what this accomplishes.

## Prerequisites

- Required permissions
- What you need to know beforehand

## Workflow

1. **Step name**:
   ```
   search_api(operationId="Service/Method")
   ```
   ```
   call_api(operationId="Service/Method", body={...})
   ```

2. **Next step**:
   ...

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| error message | why it happens | how to fix |
```

### Guidelines

1. **Always include `search_api` before `call_api`** - Claude needs to fetch the schema to know exact field names

2. **Use placeholders consistently**:
   - `{project-id}` - project identifier
   - `{instance-id}` - instance identifier
   - `{database-name}` - database name
   - `{sheet-id}` - sheet identifier
   - `{plan-id}` - plan identifier

3. **Note encoding requirements** - e.g., sheet content is base64 encoded

4. **List required permissions** in Prerequisites

5. **Include common errors** to help with troubleshooting

### Adding a New Skill

1. Create `skills/new-skill.md` with frontmatter and content
2. Run tests: `go test ./backend/api/mcp/... -run TestGetSkill`
3. The skill is automatically discovered via `//go:embed`

### Testing

```bash
# Run all skill tests
go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestGetSkill

# Verify skill loads correctly
go test -v github.com/bytebase/bytebase/backend/api/mcp -run ^TestGetSkillAllSkillsLoadable
```

## Tool Description

When adding new skills, update the tool description in `tool_skill.go`:

```go
const getSkillDescription = `Get step-by-step guides for Bytebase tasks.

**Workflow:** get_skill("task") → search_api(operationId) → call_api(...)

Skills: execute-sql, create-instance, create-project, schema-change, NEW-SKILL`
```
