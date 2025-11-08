# Button Spacing Standardization

## Overview

This document defines the spacing standards for button groups and general layout spacing in the Bytebase frontend codebase.

## Spacing Standards

### Standard Spacing Value

**ONE STANDARD TO RULE THEM ALL:**

| Value     | Pixels | Use Case                                                          |
| --------- | ------ | ----------------------------------------------------------------- |
| `gap-x-2` | 8px    | **ALL button groups** (modals, drawers, toolbars, inline actions) |

**Optional (Rare cases only):**

| Value     | Pixels | Use Case                                      |
| --------- | ------ | --------------------------------------------- |
| `gap-x-1` | 4px    | Extremely tight spacing (dense icon toolbars) |

### Why This Simplification?

- ✅ **Easier to remember**: One standard for all button groups
- ✅ **Easier to maintain**: No need to decide between gap-x-2 vs gap-x-3
- ✅ **Consistent UX**: Uniform spacing across the entire application
- ✅ **Faster implementation**: Less decision-making during development
- ✅ **Simpler code reviews**: Clear yes/no - is it gap-x-2?

## Usage Guidelines

### Modal/Drawer Footers

```vue
<!-- ✅ CORRECT: Use gap-x-2 for modal/drawer footers -->
<div class="flex justify-end gap-x-2">
  <NButton @click="cancel">Cancel</NButton>
  <NButton type="primary" @click="confirm">Confirm</NButton>
</div>

<!-- ❌ WRONG: Never use space-x -->
<div class="flex justify-end space-x-2">
  <NButton @click="cancel">Cancel</NButton>
  <NButton type="primary" @click="confirm">Confirm</NButton>
</div>
```

### Inline Action Buttons

```vue
<!-- ✅ CORRECT: Use gap-x-2 for inline action buttons -->
<div class="flex items-center gap-x-2">
  <NButton>Edit</NButton>
  <NButton>Delete</NButton>
</div>
```

### Toolbar Actions

```vue
<!-- ✅ CORRECT: Use gap-x-2 for toolbar buttons -->
<div class="flex items-center gap-x-2">
  <NButton>Export</NButton>
  <NButton>Import</NButton>
  <NButton type="primary">Create</NButton>
</div>
```

### Vertical Spacing

```vue
<!-- ✅ CORRECT: Use flex flex-col with gap-y-* -->
<div class="flex flex-col gap-y-4">
  <div>Section 1</div>
  <div>Section 2</div>
  <div>Section 3</div>
</div>

<!-- ❌ WRONG: Never use space-y -->
<div class="space-y-4">
  <div>Section 1</div>
  <div>Section 2</div>
</div>
```

### Icon Button Groups

```vue
<!-- ✅ PREFERRED: Use gap-x-2 even for icons when possible -->
<div class="flex items-center gap-x-2">
  <NButton quaternary circle><Icon /></NButton>
  <NButton quaternary circle><Icon /></NButton>
</div>

<!-- ⚠️ OPTIONAL: Use gap-x-1 only when space is critical -->
<div class="flex items-center gap-x-1">
  <NButton quaternary circle><Icon /></NButton>
  <NButton quaternary circle><Icon /></NButton>
</div>
```

## Important Notes

### Always Add Flex Direction

When using `gap-*` utilities, always include the appropriate flex direction:

```vue
<!-- ✅ CORRECT: Horizontal spacing with flex -->
<div class="flex gap-x-2">...</div>

<!-- ✅ CORRECT: Vertical spacing with flex flex-col -->
<div class="flex flex-col gap-y-4">...</div>

<!-- ❌ WRONG: gap-* without flex context -->
<div class="gap-x-2">...</div>
```

### Grid Layouts

`gap-*` works with both flexbox and grid:

```vue
<!-- ✅ Works with grid too -->
<div class="grid grid-cols-3 gap-4">
  <div>Item 1</div>
  <div>Item 2</div>
  <div>Item 3</div>
</div>
```

### Wrapping Layouts

`gap-*` handles wrapped items correctly:

```vue
<!-- ✅ CORRECT: gap-* works perfectly with flex-wrap -->
<div class="flex flex-wrap gap-x-2 gap-y-2">
  <NButton>Button 1</NButton>
  <NButton>Button 2</NButton>
  <NButton>Button 3</NButton>
  <!-- Buttons wrap correctly with consistent spacing -->
</div>

<!-- ❌ WRONG: space-* breaks when items wrap -->
<div class="flex flex-wrap space-x-2">
  <!-- Spacing breaks on wrapped lines -->
</div>
```

## Code Review Checklist

When reviewing code or creating new components, verify:

- ✅ Uses `gap-x-2` for button groups
- ✅ Uses `gap-x-*` or `gap-y-*` for ALL spacing needs
- ✅ Has `flex` or `flex flex-col` when using `gap-*`
- ❌ **REJECT** any use of `space-x-*` or `space-y-*`

## Exceptions

The following cases may deviate from the standard:

1. **Third-party components** - Don't modify vendored code
2. **Documented tight spacing** - If using `gap-x-1`, document why
3. **Responsive designs** - May use different spacing at breakpoints (e.g., `gap-x-2 md:gap-x-4`)

## Quick Reference

### Do This ✅

```vue
<!-- Horizontal button group -->
<div class="flex gap-x-2">
  <NButton>Cancel</NButton>
  <NButton type="primary">Confirm</NButton>
</div>

<!-- Vertical stack -->
<div class="flex flex-col gap-y-4">
  <div>Item 1</div>
  <div>Item 2</div>
</div>

<!-- Wrapped layout -->
<div class="flex flex-wrap gap-2">
  <NBadge />
  <NBadge />
  <NBadge />
</div>
```

### NOT This ❌

```vue
<!-- NEVER use space-x or space-y -->
<div class="flex space-x-2">
  <NButton>Cancel</NButton>
  <NButton>Confirm</NButton>
</div>

<div class="space-y-4">
  <div>Item 1</div>
  <div>Item 2</div>
</div>
```

## Verification

To check for violations in your code:

```bash
# Should return no results
grep -r "space-[xy]-[0-9]" src/components --include="*.vue"
```

If you find `space-*` utilities, replace them with `gap-*` immediately.
