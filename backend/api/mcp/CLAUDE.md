# MCP Server Skills

This directory contains the Bytebase MCP server implementation with embedded skills that provide step-by-step guides for common tasks.

## Writing Skills

Skills are markdown files in `skills/` that get embedded into the binary and served via the `get_skill` tool.

### File Structure

```
skills/
├── query.md
├── database-change.md
└── grant-permission.md
```

### Skill Format

Each skill file must have YAML frontmatter followed by markdown content:

```markdown
---
name: skill-name
description: Use when [triggering conditions] - [what it does]
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

1. **Description starts with "Use when..."** - Include triggering conditions and keywords for discoverability

2. **Always include `search_api` before `call_api`** - Claude needs to fetch the schema to know exact field names

3. **Use placeholders consistently**:
   - `{project-id}` - project identifier
   - `{instance-id}` - instance identifier
   - `{database-name}` - database name
   - `{sheet-id}` - sheet identifier
   - `{plan-id}` - plan identifier

4. **Prefer discovery over hardcoding** - Use `search_api(schema="EnumName")` instead of listing enum values

5. **Note encoding requirements** - e.g., sheet content is base64 encoded

6. **List required permissions** in Prerequisites

7. **Include common errors** to help with troubleshooting

### Adding a New Skill

1. Create `skills/new-skill.md` with frontmatter and content
2. Update `tool_skill.go` to add the skill name to `getSkillDescription`
3. Update `tool_skill_test.go` to include the skill in `TestGetSkillAllSkillsLoadable`
4. Run tests: `go test ./backend/api/mcp/... -run TestGetSkill`

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

Skills: query, database-change, grant-permission, NEW-SKILL`
```
