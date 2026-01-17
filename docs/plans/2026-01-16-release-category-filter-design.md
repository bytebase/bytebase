# Release Category Filter Design

**Date:** 2026-01-16
**Author:** Claude
**Status:** Implementation

## Overview

Add category-based filtering to the Release dashboard to help users separate releases targeting different database schemas (e.g., webapp releases vs analytics releases) within the same project.

## Background

In GitOps workflows, users often have multiple database groups with different schemas in the same project:
- `webapp` databases with application schema
- `analytics` databases with analytics schema

Users create separate Bytebase action workflows that create releases with different naming prefixes:
- `webapp-001`, `webapp-002`, etc.
- `analytics-001`, `analytics-002`, etc.

Currently, all releases are mixed together in the release dashboard, making it difficult to:
- View only webapp-related releases
- View only analytics-related releases
- Distinguish release types at a glance

## Goals

1. Allow releases to have an explicit `category` field
2. Filter releases by category in the UI
3. Support URL-based filtering (bookmarkable, shareable)
4. Maintain backward compatibility with existing releases

## Non-Goals

- Rollout dashboard filtering (deferred to future work)
- Multi-category filtering (single category only)
- Category management UI (categories are derived from release names)
- Category validation rules (flexible, set by Bytebase action)

## Design

### Architecture

**Data Flow:**
1. Bytebase action extracts category from release name prefix (e.g., "webapp" from "webapp-001")
2. Action sets `category` field when calling `CreateRelease` API
3. Backend stores category in database
4. Frontend fetches available categories via `ListReleaseCategories` API
5. User selects category from dropdown
6. Frontend updates URL and fetches filtered releases via `ListReleases` with CEL filter

### Proto Changes

#### 1. Add category to Release

```protobuf
message Release {
  string name = 1;

  // Category extracted from release name (e.g., "webapp", "analytics")
  // Set by Bytebase action during release creation
  string category = 2;

  repeated File files = 3;
  // ... other fields
}
```

#### 2. Add filter to ListReleasesRequest

```protobuf
message ListReleasesRequest {
  string parent = 1;
  int32 page_size = 2;
  string page_token = 3;
  bool show_deleted = 4;

  // Filter expression using CEL
  // Supported filters:
  // - category: release category, support "==" operator only.
  //
  // Example:
  // category == "webapp"
  string filter = 5;
}
```

#### 3. Add ListReleaseCategories RPC

```protobuf
service ReleaseService {
  // ... existing methods ...

  // Lists all unique categories in a project
  // Permissions required: bb.releases.list
  rpc ListReleaseCategories(ListReleaseCategoriesRequest) returns (ListReleaseCategoriesResponse) {
    option (google.api.http) = {get: "/v1/{parent=projects/*}/releases:listCategories"};
    option (google.api.method_signature) = "parent";
    option (bytebase.v1.permission) = "bb.releases.list";
    option (bytebase.v1.auth_method) = IAM;
  }
}

message ListReleaseCategoriesRequest {
  string parent = 1 [(google.api.field_behavior) = REQUIRED];
}

message ListReleaseCategoriesResponse {
  repeated string categories = 1;
}
```

### Database Schema

**Migration:**
```sql
ALTER TABLE release ADD COLUMN category TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_release_category ON release(project_id, category) WHERE row_status = 'NORMAL';
```

**Index rationale:**
- Filtering by category is a common query pattern
- Index on (project_id, category) enables efficient lookups
- Partial index excludes deleted releases (row_status = 'NORMAL')

### Backend Implementation

#### Store Layer

**ReleaseMessage:**
```go
type ReleaseMessage struct {
    UID         string
    ProjectID   string
    Category    string  // NEW
    // ... other fields
}

type FindReleaseMessage struct {
    ProjectID  *string
    Category   *string  // NEW: Single category filter
    // ... other fields
}
```

**New method:**
```go
func (s *Store) ListReleaseCategories(ctx context.Context, projectResourceID string) ([]string, error)
```

#### API Service Layer

**CEL Filter Parser:**
- Parse `category == "value"` expressions
- Extract category value
- Return error for unsupported operators

**ListReleaseCategories handler:**
- Validate project access
- Query distinct categories from database
- Return sorted list

### Frontend Implementation

#### URL Structure

- Single category: `?category=webapp`
- No category (all): `/releases` (no query param)

#### UI Components

**Filter toolbar:**
```
┌─────────────────────────────────────────────────────────┐
│ Releases                                           [+]   │
├─────────────────────────────────────────────────────────┤
│ Category: [All ▼] [webapp] [analytics]                  │
├─────────────────────────────────────────────────────────┤
│ Name          │ Files       │ Created                   │
```

**Implementation:**
- Single-select dropdown (NSelect)
- "All" option shows all releases (no filter)
- URL query param synced with selection
- Browser back/forward navigation supported

#### Files to Modify

- `frontend/src/utils/releaseFilter.ts` - URL/filter conversion utilities
- `frontend/src/composables/useReleaseCategories.ts` - Category fetching
- Release list view (TBD: find correct file) - Filter UI
- `frontend/src/locales/en-US.json` - i18n strings

### Bytebase Action Integration

The Bytebase GitHub Action will:

```go
func createRelease(opts *CreateOptions) error {
    // Extract category from release name (before first hyphen)
    category := extractCategory(opts.ReleaseName)

    request := &v1pb.CreateReleaseRequest{
        Release: &v1pb.Release{
            Category: category,
            // ... other fields
        },
    }
}

func extractCategory(releaseName string) string {
    parts := strings.SplitN(releaseName, "-", 2)
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}
```

## Migration Strategy

### Existing Releases

Releases created before this feature will have `category = ""` (empty string).

**Options:**
1. Show as "Uncategorized" in UI
2. Provide backfill script to extract category from existing release names
3. Leave empty and let them age out

**Recommendation:** Option 1 (show as "Uncategorized") - simplest, no data migration needed.

### Rollout

1. Deploy backend changes (database migration + API)
2. Deploy frontend changes (UI updates)
3. Update Bytebase action to set category
4. Document category extraction logic for users

## Testing

### Unit Tests

- CEL filter parser tests
- Store layer tests (category filtering, listing)
- Frontend utility tests (URL conversion)

### Integration Tests

- End-to-end release creation with category
- Category listing
- Release filtering

### Manual Testing

- Create releases with categories
- Filter by category
- Verify URL updates
- Test browser navigation
- Test with empty categories
- Test with special characters

## Alternatives Considered

### 1. Infer from Database Group

**Approach:** Store target database group in Release, filter by group

**Rejected because:**
- Requires more proto changes
- Less flexible (what if release targets multiple groups?)
- Category is more explicit and clear

### 2. Pattern-Based Auto-Categorization

**Approach:** Configure regex patterns in project settings

**Rejected because:**
- Brittle (breaks if naming changes)
- Complex configuration
- Less explicit than setting category directly

### 3. Multi-Category Filtering

**Approach:** Support `category in ["webapp", "analytics"]`

**Rejected because:**
- More complex UI (multi-select)
- More complex CEL parsing
- Single category filter meets current requirements

## Future Enhancements

1. **Rollout dashboard filtering** - Add category to Rollout proto, filter rollouts
2. **Category management** - Bulk rename, merge categories
3. **Category metadata** - Description, color, icon
4. **Multi-category filter** - If users need to view multiple categories simultaneously
5. **Category-based permissions** - Restrict access by category

## Success Metrics

- Users can filter releases by category
- URL-based filtering works (shareable links)
- No performance regression (filter query < 100ms)
- Zero breaking changes for existing users

## References

- Similar pattern: `ListRolloutsRequest.filter` field
- CEL documentation: https://github.com/google/cel-spec
- Google AIP-160: Filtering
