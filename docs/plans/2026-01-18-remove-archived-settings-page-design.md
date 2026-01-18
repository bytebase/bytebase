# Remove Archived Settings Page Design

## Overview

Remove the dedicated "Archived" settings page (`/setting/archive`) and integrate archived projects and instances into their respective list pages with state filtering.

## Background

Currently, archived (soft-deleted) projects and instances are only accessible through Settings → Archived. This creates a separate location for managing archived resources, adding navigation overhead and separating related functionality.

## Goals

- Consolidate project management to the Projects page
- Consolidate instance management to the Instances page
- Reduce navigation depth for accessing archived items
- Maintain all existing functionality (restore, hard delete)

## Design

### 1. State Filter Integration

Add a "State" filter dropdown to both Projects and Instances list pages with three options:
- **Active** (default) - shows only active items
- **Archived** - shows only deleted/archived items
- **All** - shows both active and archived items

**Location:** Place the state filter in the toolbar area near other filters (search box)

**Default behavior:** Default to "Active" state to show only active items

**Permissions:** Only show "Archived" option if user has undelete permission:
- Projects: `bb.projects.undelete`
- Instances: `bb.instances.undelete`

### 2. Visual Differentiation

Archived items displayed in the list should be visually distinct:
- Reduced opacity (e.g., `opacity-60`)
- Small "Archived" badge or tag next to the name
- Subtle background tint (optional)

### 3. Actions and Behavior

**Available actions for archived items:**
- Restore button (if user has undelete permission)
- Hard delete button (if user has delete permission)
- Archive button hidden (already archived)

**Empty states:**
- "No archived projects" when Archived filter selected but no results
- "No archived instances" when Archived filter selected but no results
- Different from active items empty state

### 4. Navigation Changes

**Remove archive settings route:**
- Delete route definition in `frontend/src/router/dashboard/workspaceSetting.ts`
- Remove `SETTING_ROUTE_WORKSPACE_ARCHIVE` export (line 10)
- Remove archive route configuration (lines 60-72)
- Remove "Archived" menu item from settings sidebar

**Update hard-delete redirects:**
- `ProjectArchiveRestoreButton.vue`: redirect to Projects list page with "Archived" filter active
- `InstanceArchiveRestoreButton.vue`: redirect to Instances list page with "Archived" filter active

**Delete archive page:**
- Remove `frontend/src/views/Archive.vue` entirely

## Implementation

### Files to Modify

1. **Routing:**
   - `frontend/src/router/dashboard/workspaceSetting.ts` - remove archive route
   - Settings sidebar configuration - remove "Archived" menu item

2. **List pages:**
   - Projects list page - add state filter dropdown
   - Instances list page - add state filter dropdown
   - Wire state selection to `PagedProjectTable` and `PagedInstanceTable` filter props

3. **Archive/Restore components:**
   - `frontend/src/components/Project/ProjectArchiveRestoreButton.vue` - update hard-delete redirect (line 76-82)
   - `frontend/src/components/Instance/InstanceArchiveRestoreButton.vue` - update hard-delete redirect (line 112-117)

4. **Table components:**
   - Add visual styling for archived rows
   - Update empty state messages

5. **Cleanup:**
   - Delete `frontend/src/views/Archive.vue`

### Technical Details

**Existing filter support:**
Both `PagedProjectTable` and `PagedInstanceTable` already support state filtering via the `filter` prop:
- `State.ACTIVE` - active items only
- `State.DELETED` - archived items only
- `undefined` - all items

The components handle this filtering at the API level, so no backend changes are required.

## Benefits

- **Consolidated functionality:** All project management in one place, all instance management in one place
- **Reduced navigation depth:** No need to navigate to Settings → Archived
- **Familiar patterns:** Filtering is consistent with existing UI patterns
- **Better discoverability:** Users naturally look in list pages for items
- **Less maintenance:** One fewer specialized page to maintain

## Migration Notes

- No data migration required
- No API changes required
- Purely frontend routing and UI changes
- Existing permissions model remains unchanged
