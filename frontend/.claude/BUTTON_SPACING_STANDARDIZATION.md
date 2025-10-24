# Button Spacing Standardization Plan

## Executive Summary

This document outlines the plan to standardize button spacing across the Bytebase frontend codebase. Currently, button spacing is inconsistent, using both `space-x` and `gap-x` utilities with varying values (1-6).

## Current State Analysis

### Statistics
- **Total occurrences**: 616+ spacing utilities in Vue files
  - `space-x-*`: 199 occurrences across 134 files
  - `gap-x-*`: 417 occurrences across 255 files

### Issues Identified

1. **Mixed Utilities**: Both `space-x` and `gap-x` used interchangeably
2. **Inconsistent Values**: Multiple spacing values for similar contexts
3. **No Clear Standards**: No documented guidelines for when to use which spacing

### Common Patterns Found

**Modal/Drawer Footers:**
- `space-x-2` (older components)
- `gap-x-3` (newer components)
- `space-x-3` (mixed)

**Toolbar Actions:**
- `space-x-2`
- `gap-x-2`

**Inline Button Groups:**
- Values range from `1` to `4`

## Proposed Standards

### 1. Utility Choice: Use `gap-x` Over `space-x`

**Rationale:**
- `gap-x` works better with flexbox and grid layouts
- `gap-x` doesn't add spacing to the last child (cleaner)
- Newer components already prefer `gap-x`
- Better browser support for modern CSS

### 2. Simplified Spacing Standard

**ONE STANDARD TO RULE THEM ALL:**

| Value | Pixels | Use Case |
|-------|--------|----------|
| `gap-x-2` | 8px | **ALL button groups** (modals, drawers, toolbars, inline actions) |

**Optional (Rare cases only - use sparingly):**

| Value | Pixels | Use Case | Example |
|-------|--------|----------|---------|
| `gap-x-1` | 4px | Extremely tight spacing when space is critical | Dense icon toolbars (rare) |

**Why This Simplification?**
- ✅ **Easier to remember**: One standard for all button groups
- ✅ **Easier to maintain**: No need to decide between gap-x-2 vs gap-x-3
- ✅ **Consistent UX**: Uniform spacing across the entire application
- ✅ **Faster implementation**: Less decision-making during migration
- ✅ **Simpler code reviews**: Clear yes/no - is it gap-x-2?

### 3. Context-Specific Guidelines

#### Modal/Drawer Footers (Action Buttons)
```vue
<!-- ✅ STANDARD: Use gap-x-2 for modal/drawer footers -->
<div class="flex justify-end gap-x-2">
  <NButton @click="cancel">Cancel</NButton>
  <NButton type="primary" @click="confirm">Confirm</NButton>
</div>
```

#### Inline Action Buttons
```vue
<!-- ✅ STANDARD: Use gap-x-2 for inline action buttons -->
<div class="flex items-center gap-x-2">
  <NButton>Edit</NButton>
  <NButton>Delete</NButton>
</div>
```

#### Toolbar Actions
```vue
<!-- ✅ STANDARD: Use gap-x-2 for toolbar buttons -->
<div class="flex items-center gap-x-2">
  <NButton>Export</NButton>
  <NButton>Import</NButton>
  <NButton type="primary">Create</NButton>
</div>
```

#### Icon Button Groups (Optional - Only When Space is Critical)
```vue
<!-- ⚠️ OPTIONAL: Use gap-x-1 only for very tight icon groups -->
<div class="flex items-center gap-x-1">
  <NButton quaternary circle><Icon /></NButton>
  <NButton quaternary circle><Icon /></NButton>
</div>

<!-- ✅ PREFERRED: Use gap-x-2 even for icons when possible -->
<div class="flex items-center gap-x-2">
  <NButton quaternary circle><Icon /></NButton>
  <NButton quaternary circle><Icon /></NButton>
</div>
```

## Exceptions & Special Cases

### When NOT to Change

1. **Third-party components** - Don't modify vendored code
2. **Intentional tight spacing** - Document why if < gap-x-1
3. **Responsive designs** - May need different spacing at breakpoints
4. **Legacy compatibility** - If component is being deprecated

### Non-Button Spacing

This plan focuses on button groups. Other spacing contexts:
- Form field spacing: Separate guideline
- List item spacing: Separate guideline
- Card spacing: Separate guideline
