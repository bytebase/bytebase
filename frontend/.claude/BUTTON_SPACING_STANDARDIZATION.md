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

## Migration Strategy

### Phase 1: High-Impact Areas (Week 1-2)
Focus on most visible and frequently used components:

**Priority 1 - Modals & Drawers:**
- All modal footers → `gap-x-2`
- All drawer footers → `gap-x-2`
- Estimated: ~50 files

**Priority 2 - Common Components:**
- Feature modals
- Settings panels
- Form action buttons
- Estimated: ~30 files

**Files to update:**
```
frontend/src/components/IAMRemindModal.vue
frontend/src/components/ReleaseRemindModal.vue
frontend/src/components/FeatureGuard/FeatureModal.vue
frontend/src/views/SchemaTemplate/ColumnTypesUpdateFailedModal.vue
frontend/src/components/FileContentPreviewModal.vue
... (see detailed file list in appendix)
```

### Phase 2: Medium-Impact Areas (Week 3-4)
**Forms and Settings:**
- Database forms
- Instance forms
- Project settings
- Estimated: ~60 files

**Table Actions:**
- Data table toolbars
- Inline row actions
- Estimated: ~40 files

### Phase 3: Low-Impact Areas (Week 5-6)
**Specialized Components:**
- SQL Editor panels
- Schema editor
- Diagram components
- AI components
- Estimated: ~80 files

### Phase 4: Edge Cases & Cleanup (Week 7)
- Review all changes
- Fix any missed instances
- Update any new code

## Implementation Guidelines

### 1. Search & Replace Pattern

**DO NOT use global search/replace**. Review each instance:

```bash
# Find all space-x in Vue files
grep -r "space-x-" frontend/src/**/*.vue

# Find specific patterns for button groups
grep -r "justify-end.*space-x" frontend/src/**/*.vue
```

### 2. Manual Review Checklist

For each file:
- [ ] Identify button group context
- [ ] Change `space-x-*` to `gap-x-2` for button groups
- [ ] Test visual appearance
- [ ] Check responsive behavior

### 3. Code Review Standards

**Before:**
```vue
<div class="mt-7 flex justify-end space-x-2">
  <NButton @click="cancel">Cancel</NButton>
  <NButton type="primary" @click="confirm">OK</NButton>
</div>
```

**After:**
```vue
<div class="mt-7 flex justify-end gap-x-2">
  <NButton @click="cancel">Cancel</NButton>
  <NButton type="primary" @click="confirm">OK</NButton>
</div>
```

### 4. Testing Requirements

For each updated component:
1. Visual regression test (screenshot comparison)
2. Responsive behavior check (mobile, tablet, desktop)
3. Dark mode check (if applicable)

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

## Documentation Updates

After migration:
1. Update component library documentation
2. Add to style guide
3. Create ESLint rule (if possible)
4. Add to CLAUDE.md for AI assistant guidance

## Success Metrics

- [ ] 100% of button groups use `gap-x-2`
- [ ] Zero usage of `space-x` for button groups
- [ ] < 5% exceptions (documented with gap-x-1)
- [ ] All new code follows standards

## Timeline

| Phase | Duration | Completion Date |
|-------|----------|-----------------|
| Phase 1 | 2 weeks | Week 2 |
| Phase 2 | 2 weeks | Week 4 |
| Phase 3 | 2 weeks | Week 6 |
| Phase 4 | 1 week | Week 7 |
| **Total** | **7 weeks** | **End of Week 7** |

## Resources Required

- Developer time: 1-2 developers, 50% allocation
- QA time: Visual regression testing
- Design review: Final approval on spacing values

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Visual regressions | High | Screenshot testing, staged rollout |
| Breaking changes | Medium | Thorough testing, gradual migration |
| Inconsistent adoption | High | Clear documentation, code review |
| Developer resistance | Low | Show examples, document rationale |

## Appendix A: High-Priority File List

### Modal Footers (Space-x → Gap-x-2)
```
frontend/src/components/IAMRemindModal.vue (line 25)
frontend/src/components/ReleaseRemindModal.vue (line 19)
frontend/src/components/FeatureGuard/FeatureModal.vue (line 61)
frontend/src/views/SchemaTemplate/ColumnTypesUpdateFailedModal.vue (line 31)
frontend/src/components/FileContentPreviewModal.vue (line 23)
frontend/src/components/UploadFilesButton.vue (line 34)
frontend/src/views/auth/InactiveRemindModal.vue (line 23)
frontend/src/components/AlterSchemaPrepForm/SchemaEditorModal.vue (lines 51, 88)
```

### Drawer Footers (Update to Gap-x-2)
```
frontend/src/components/LabelEditorDrawer.vue (line 15) - change gap-x-3 to gap-x-2
frontend/src/components/User/Settings/CreateGroupDrawer.vue (line 114) - change gap-x-3 to gap-x-2
frontend/src/components/User/Settings/CreateUserDrawer.vue (line 99) - change gap-x-3 to gap-x-2
frontend/src/components/User/Settings/AADSyncDrawer.vue (line 104) - change gap-x-3 to gap-x-2
```

### Toolbar Actions (Space-x → Gap-x-2)
```
frontend/src/views/SettingWorkspaceSQLReview.vue (line 15)
frontend/src/views/SettingWorkspaceIM.vue (line 30)
frontend/src/components/Database/DatabaseChangelogPanel.vue (line 15)
frontend/src/components/Role/Setting/components/RoleDataTable/cells/RoleOperationsCell.vue (line 2)
```

## Appendix B: Quick Reference

### Decision Tree

```
Is this a button group?
├─ Yes → Use gap-x-2 ✅
└─ No → Not covered by this standard

Exception: Is space EXTREMELY tight?
├─ Yes → Consider gap-x-1 (rare, needs justification)
└─ No → Use gap-x-2 ✅
```

**That's it! Super simple.**

## Appendix C: ESLint Rule (Future)

```js
// Potential ESLint rule to enforce standards
// .eslintrc.js or custom plugin
{
  'bytebase/button-spacing': ['error', {
    'utility': 'gap-x',
    'disallow': ['space-x'],
    'standard': 'gap-x-2', // One standard for all!
    'allowExceptions': ['gap-x-1'] // Rare cases only
  }]
}
```

---

**Document Version:** 1.0
**Last Updated:** 2025-10-24
**Owner:** Frontend Team
**Reviewers:** Design Team, Tech Lead
