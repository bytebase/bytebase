# Fix resolveDatabase Permission Model

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix `resolveDatabase` to work with standard user permissions (WorkspaceMember) by listing databases through project-scoped API calls instead of workspace-scoped.

**Architecture:** Replace the single `workspaces/-` ListDatabases call with: list user's projects → list databases per project with existing CEL filter. This respects Bytebase's RBAC model where most users have project-level access, not workspace-level `bb.databases.list`.

**Tech Stack:** Go, existing MCP server infrastructure

---

## Problem

`resolveDatabase` calls `ListDatabases` with `parent: "workspaces/-"`. This fails because:
1. `workspaces/-` is not a valid wildcard — `-` doesn't match the real workspace ID
2. Even with the correct workspace ID, the `WorkspaceMember` role (default for new/OAuth users) doesn't have `bb.databases.list` at workspace scope
3. Users access databases through project memberships, not workspace-level permissions

## Solution

Replace the single ListDatabases call with a two-step approach:
1. Call `ProjectService/ListProjects` (no special permissions needed — returns projects the user has access to)
2. For each project, call `DatabaseService/ListDatabases` with `parent: "projects/{id}"` and the existing CEL filter

This is N+1 calls but N is small (few projects) and the server-side filter keeps responses minimal. If the user provides a `project` input param, skip step 1 and query that project directly.

---

### Task 1: Refactor resolveDatabase to use project-scoped queries

**Files:**
- Modify: `backend/api/mcp/tool_query.go`

**Step 1: Add `listProjects` helper method**

```go
// listProjects returns project resource names the user has access to.
func (s *Server) listProjects(ctx context.Context) ([]string, error) {
    resp, err := s.apiRequest(ctx, "/bytebase.v1.ProjectService/ListProjects", map[string]any{})
    if err != nil {
        return nil, errors.Wrap(err, "failed to list projects")
    }
    if resp.Status >= 400 {
        return nil, errors.Errorf("failed to list projects: %s", parseError(resp.Body))
    }
    var result struct {
        Projects []struct {
            Name string `json:"name"`
        } `json:"projects"`
    }
    if err := json.Unmarshal(resp.Body, &result); err != nil {
        return nil, errors.Wrap(err, "failed to parse project list")
    }
    names := make([]string, 0, len(result.Projects))
    for _, p := range result.Projects {
        names = append(names, p.Name)
    }
    return names, nil
}
```

**Step 2: Add `listDatabasesInProject` helper method**

```go
// listDatabasesInProject returns databases matching the filter in a project.
func (s *Server) listDatabasesInProject(ctx context.Context, project, filter string) ([]databaseEntry, error) {
    body := map[string]any{
        "parent":   project,
        "filter":   filter,
        "pageSize": 1000,
    }
    resp, err := s.apiRequest(ctx, "/bytebase.v1.DatabaseService/ListDatabases", body)
    if err != nil {
        return nil, err
    }
    // Permission denied on a project is not fatal — skip it.
    if resp.Status >= 400 {
        return nil, nil
    }
    var listResp listDatabasesResponse
    if err := json.Unmarshal(resp.Body, &listResp); err != nil {
        return nil, err
    }
    return listResp.Databases, nil
}
```

**Step 3: Refactor `resolveDatabase`**

Replace the body of `resolveDatabase` from the `workspaces/-` call through the `listResp` parsing with:

```go
func (s *Server) resolveDatabase(ctx context.Context, input QueryInput) (*resolvedDatabase, error) {
    filter := buildDatabaseFilter(input)

    var allDatabases []databaseEntry

    if input.Project != "" {
        // User specified a project — query it directly.
        databases, err := s.listDatabasesInProject(ctx, "projects/"+input.Project, filter)
        if err != nil {
            return nil, errors.Wrap(err, "failed to list databases")
        }
        allDatabases = databases
    } else {
        // List user's projects, then query each for matching databases.
        projects, err := s.listProjects(ctx)
        if err != nil {
            return nil, err
        }
        for _, project := range projects {
            databases, err := s.listDatabasesInProject(ctx, project, filter)
            if err != nil {
                return nil, errors.Wrap(err, "failed to list databases")
            }
            allDatabases = append(allDatabases, databases...)
        }
    }

    // ... rest of tiered matching unchanged ...
}
```

**Step 4: Remove `project` from `buildDatabaseFilter`**

Since project filtering is now handled by the `parent` field (querying per-project), remove the `project` filter clause from `buildDatabaseFilter`. Keep `instance` filter.

**Step 5: Run tests, lint, verify**

Run: `go test -count=1 ./backend/api/mcp/...`
Run: `golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: Tests will fail because the mock server doesn't handle ListProjects. Fix in Task 2.

---

### Task 2: Update test mocks for project-scoped resolution

**Files:**
- Modify: `backend/api/mcp/tool_query_test.go`

**Step 1: Update `mockListDatabases` to handle both endpoints**

The mock needs to serve `ListProjects` (returns a single test project) and `ListDatabases` (existing filter-aware logic), routing by URL path.

```go
func mockListDatabases(databases []map[string]any) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        switch r.URL.Path {
        case "/bytebase.v1.ProjectService/ListProjects":
            _ = json.NewEncoder(w).Encode(map[string]any{
                "projects": []map[string]any{{"name": "projects/test-project"}},
            })
        case "/bytebase.v1.DatabaseService/ListDatabases":
            // existing filter logic unchanged
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    })
}
```

**Step 2: Update `mockQueryServer` if needed**

`mockQueryServer` delegates to `mockListDatabases` for non-Query paths, so it should work automatically once `mockListDatabases` is updated.

**Step 3: Run all tests and lint**

Run: `go test -v -count=1 ./backend/api/mcp/...`
Run: `golangci-lint run --allow-parallel-runners ./backend/api/mcp/...`

Expected: All tests pass, 0 lint issues.

---

### Task 3: Commit, push, verify CI

**Step 1: Commit**

```
fix(mcp): use project-scoped queries for database resolution

The default WorkspaceMember role doesn't have bb.databases.list
at workspace scope. List user's projects first, then query databases
per project with the existing CEL filter.
```

**Step 2: Push and verify CI**

Run: `git push`
Check: golangci-lint, SonarCloud, go-tests all pass.

---

## Summary of changes

| Action | File |
|--------|------|
| Modify | `backend/api/mcp/tool_query.go` — add `listProjects`, `listDatabasesInProject`, refactor `resolveDatabase`, update `buildDatabaseFilter` |
| Modify | `backend/api/mcp/tool_query_test.go` — update mocks to handle ListProjects endpoint |
