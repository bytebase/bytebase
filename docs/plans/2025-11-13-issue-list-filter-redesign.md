# Issue List Filter Redesign - GitHub Style

**Date:** 2025-11-13
**Status:** Design Complete

## Overview

Redesign the issue list page filter and search interface to follow GitHub's pattern: an always-visible advanced search bar with quick action buttons below that mutate the filter string.

## Goals

- Remove the simple/advanced mode toggle - make the advanced search always visible
- Add GitHub-style quick action buttons that modify the search string
- Distinguish between preset filters (mutually exclusive) and toggleable filters (additive)
- Maintain URL query parameter sync for bookmarking/sharing
- Keep the existing tag-based filter display in the search bar

## Layout Structure

```
┌─────────────────────────────────────────────────────────────┐
│ 🔍 Filter: [tag] [tag] [search text...]              [×]    │ ← Advanced Search Bar
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ Row 1: [Waiting Approval] [Created] [All]                   │ ← Preset Buttons
│ Row 2: Status ▾ | Creator ▾ | Assignee ▾ | Project ▾ | ... │ ← Toggle Filter Dropdowns
└─────────────────────────────────────────────────────────────┘
```

### Key Changes from Current Design

1. **Remove the toggle** - Advanced search bar is always visible, no "simple mode" vs "advanced mode"
2. **Remove the tabs component** - Replace `TabFilter` with preset buttons
3. **Add filter dropdown buttons** - New row of dropdown buttons for toggling filters
4. **Keep tag-based display** - Filter tags remain as removable chips inside the search input

## Component Breakdown

### Always Visible Components

- **AdvancedSearch component** - Stays mostly the same, always rendered
- **PresetButtons component** (NEW) - Renders "Waiting Approval | Created | All"
- **FilterToggles component** (NEW) - Renders "Status ▾ | Creator ▾ | ..."
- **IssueSearch component** (MODIFIED) - Remove toggle logic, compose the new layout

## Preset Buttons (Row 1)

### Visual Design

- Three buttons in a button group: `[Waiting Approval] [Created] [All]`
- Visual style: Similar to current `TabFilter` but rendered as buttons
- States:
  - **Active**: One preset always highlighted (or none if custom search)
  - **Inactive**: Other presets clickable but not highlighted
  - Mutually exclusive - clicking one deselects others

### Behavior & Search String Mapping

Preset buttons **replace** the search params:

**Waiting Approval:**
- Sets: `status:OPEN approval:pending approver:{current-user-email}`
- Visual tags: `[status:OPEN] [approval:pending] [approver:me]`
- Clears any other non-readonly scopes

**Created:**
- Sets: `creator:{current-user-email}`
- Visual tags: `[creator:me]`
- Clears other scopes (including status - shows all statuses)

**All:**
- Clears all filters except readonly scopes (like `project:xxx` on project page)
- Functions as a complete reset

### Active Preset Detection

Reuse existing `guessTabValueFromSearchParams` logic:
- Analyze current `SearchParams`
- Match against preset patterns
- Highlight matching preset button
- If no match, no button highlighted (custom search state)

## Toggle Filter Dropdowns (Row 2)

### Visual Design

- Horizontal arrangement: `Status ▾ | Creator ▾ | Assignee ▾ | Project ▾ | Database ▾ | ...`
- Button states:
  - **Inactive**: Regular button with `▾` icon, label: `{FilterName} ▾`
  - **Active**: Highlighted button showing selected value, label: `{FilterName}: {Value} ▾`
- Separator between buttons (`|` or subtle border)

### Status Dropdown (Special Behavior)

```
┌─────────────┐
│ ☐ Open      │
│ ☐ Closed    │
└─────────────┘
```

**Behavior:**
- Mutually exclusive checkboxes (only one or none can be checked)
- Checking "Open" → unchecks "Closed" → adds `status:OPEN` tag
- Checking "Closed" → unchecks "Open" → adds `status:CLOSED` tag
- Unchecking active one → removes status tag → shows all statuses

**Button Label States:**
- No selection: `Status ▾` (inactive)
- Open selected: `Status: Open ▾` (active/highlighted)
- Closed selected: `Status: Closed ▾` (active/highlighted)

### Creator/Assignee Dropdowns

```
┌──────────────────────┐
│ ☐ Me                 │
│ ──────────────────── │
│ 🔍 Search users...   │
│ ☐ user1@email.com    │
│ ☐ user2@email.com    │
└──────────────────────┘
```

- "Me" as quick toggle at top
- Search input for finding other users
- Selecting user → adds `creator:{email}` or `assignee:{email}` tag
- Multiple selections allowed if scope supports it

### Project/Database/etc. Dropdowns

```
┌──────────────────────┐
│ 🔍 Search...         │
│ ☐ Project A          │
│ ☐ Project B          │
└──────────────────────┘
```

- Searchable list
- Reuses existing scope option search functionality
- Selecting item → adds corresponding tag

### Toggle Behavior

- **Adding**: Click dropdown → select value → tag appears in search bar
- **Removing**:
  - Click × on tag in search bar
  - OR open dropdown → deselect the value
- **Changing**: Open dropdown → select different value → tag updates

## Preset & Toggle Interaction

### When Preset Clicked

1. Search params reset to preset's default filters
2. Toggle dropdowns update to reflect new state
3. Search bar shows only preset's filter tags
4. Preset button becomes highlighted

### When Toggle Used

1. If preset was active → preset becomes inactive (custom state)
2. Search enters "custom" mode (no preset highlighted)
3. User can continue adding/removing filters
4. Clicking preset resets back to that preset

### Example Flow

```
1. User clicks "Waiting Approval"
   → [status:OPEN] [approval:pending] [approver:me]
   → "Waiting Approval" highlighted
   → Status dropdown shows "Open" checked

2. User adds Creator filter via dropdown
   → [status:OPEN] [approval:pending] [approver:me] [creator:john@example.com]
   → "Waiting Approval" NO LONGER highlighted
   → All dropdowns reflect current filters

3. User clicks "All"
   → Search bar cleared (except readonly scopes)
   → "All" highlighted
   → All dropdowns reset to inactive
```

## Search Bar Behavior

### Direct Text Input

- Users can type directly to add filters manually
- Typing `status:` triggers value menu dropdown (current behavior)
- Tags removable by clicking × or backspace (current behavior)

### Interaction with Dropdowns

- Opening filter dropdown → if filter exists in search, show as selected
- Adding via dropdown → appears as tag in search bar
- Removing tag → dropdown shows as inactive

### Query Text

- After filter tags, users can type free text for searching titles/descriptions
- Example: `[status:OPEN] [creator:me] database migration`
  - Tags: `status:OPEN`, `creator:me`
  - Query: "database migration"

### URL Persistence

- `SearchParams` serialized to URL query parameters (current behavior)
- Tags are visual representation of URL state
- Supports bookmarking/sharing filtered views

## Implementation Architecture

### Component Structure

```
IssueSearch/ (modified)
├── IssueSearch.vue (simplified, no toggle logic)
├── PresetButtons.vue (NEW)
├── FilterToggles.vue (NEW)
├── FilterDropdown.vue (NEW - reusable)
└── Status.vue (modified - dropdown menu content)

AdvancedSearch/ (mostly unchanged)
├── AdvancedSearch.vue (always rendered)
└── ... (existing components)
```

### New Components

**PresetButtons.vue**
- Props: `params: SearchParams`, preset definitions
- Emits: `update:params`
- Logic: Active detection, preset selection

**FilterToggles.vue**
- Props: `params: SearchParams`, filter options
- Emits: `update:params`
- Renders: Multiple `FilterDropdown` components

**FilterDropdown.vue** (reusable)
- Props: `scopeId`, `label`, `options`, `currentValue`, `type`
- Emits: `update:value`
- Renders: Button + popover menu

### Modified Components

**IssueSearch.vue**
- Remove: `state.advanced`, `toggleAdvancedSearch()`, conditional rendering
- Add: Always render AdvancedSearch + PresetButtons + FilterToggles
- Keep: Props/emits for `params`

**MyIssues.vue & ProjectIssuesPanel.vue**
- Remove: Advanced search toggle button
- Remove: `TabFilter` usage
- Simplified: Pass params through, no mode switching

### State Management

All state in parent components:
- `state.params: SearchParams` - single source of truth
- Remove `state.advanced` toggle
- Keep watchers and localStorage logic

## Visual Design

### Spacing & Layout

- Container padding: `px-4 pb-2`
- Gap between search and buttons: `gap-y-2`
- Preset buttons: Connected button group, `h-[34px]`
- Filter toggles: `gap-x-2` between buttons

### Button States & Colors

**Preset Buttons:**
- Active: Primary background
- Inactive: Ghost/neutral style
- Hover: Subtle highlight

**Toggle Dropdowns:**
- Inactive: Ghost with `▾`
- Active: Accented + shows value
- Hover: Subtle highlight
- Dropdown icon rotates when open

### Responsive Behavior

**Desktop (md+):**
- Two full rows
- All buttons visible

**Mobile (sm-):**
- Search bar: Full width
- Preset buttons: Scrollable horizontal or stacked
- Filter toggles: Horizontal scroll or overflow menu

### Accessibility

- Proper button labels
- `aria-expanded` on dropdowns
- Keyboard navigation
- Tags keyboard-removable

### Animations

- Dropdown menus: Fade + slide (reuse existing)
- Preset switching: Smooth background transition
- Tag add/remove: Subtle fade

## Data Flow

```
User Interaction
    ↓
PresetButton or FilterDropdown
    ↓
Emit update:params
    ↓
Parent Component (MyIssues/ProjectIssuesPanel)
    ↓
Update state.params (SearchParams)
    ↓
├→ AdvancedSearch (renders tags)
├→ PresetButtons (shows active state)
├→ FilterToggles (shows active filters)
├→ URL query params (via router)
└→ Backend API (via buildIssueFilterBySearchParams)
```

## Migration Notes

### Breaking Changes

None - this is a UI reorganization with same underlying functionality.

### Backward Compatibility

- URL query parameters remain compatible
- LocalStorage keys unchanged
- API filters unchanged

### User Migration

- No migration needed
- Users may need brief adjustment to always-visible search
- Preset buttons replace familiar tabs (same functionality)

## Future Enhancements

Potential improvements not in initial scope:

1. **Saved filters** - Let users save custom filter combinations
2. **Filter suggestions** - Show common filter patterns as user types
3. **Advanced text mode** - Toggle between tags and raw text editing
4. **Filter groups** - Organize dropdowns into logical categories
5. **Recent filters** - Quick access to recently used filter combinations

## Success Metrics

- Reduced clicks to apply common filters
- Increased use of advanced filters (now more discoverable)
- Faster issue finding (preset + toggle combination)
- Positive user feedback on clarity of filter state
