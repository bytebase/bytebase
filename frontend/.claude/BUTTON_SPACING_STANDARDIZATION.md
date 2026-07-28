# Button group spacing

Use Tailwind `gap-*` utilities on the parent of every React button group. Do not use `space-x-*`, `space-y-*`, or per-button margins for sibling spacing.

```tsx
<div className="flex items-center gap-x-2">
  <Button variant="outline">Cancel</Button>
  <Button>Save</Button>
</div>
```

This applies to dialog and sheet footers, toolbars, table actions, inline actions, and responsive button rows. Use `gap-y-*` as well when a group can wrap:

```tsx
<div className="flex flex-wrap items-center gap-x-2 gap-y-2">
  <Button variant="outline">Preview</Button>
  <Button>Apply</Button>
</div>
```

Do not add an empty wrapper only to satisfy this rule. Put the gap on the existing flex or grid container that owns the sibling layout.

Audit with:

```bash
rg -n 'space-[xy]-|(?:m[lrxy]|margin)-' frontend/src --glob '*.tsx'
```

Review matches in button-group containers; not every margin elsewhere is a violation.
