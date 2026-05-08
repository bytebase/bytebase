# BYT-9390 — DDL/DML environment-grant warnings

**Status:** Approved (brainstorming)
**Linear:** [BYT-9390](https://linear.app/bytebase/issue/byt-9390/missing-dmlddl-permission-prompts-in-sql-editor-self-service)
**Author:** Peter Zhu
**Date:** 2026-05-07

## Problem

When a Bytebase project role grants `bb.sql.ddl` and/or `bb.sql.dml` and the granter (or requester) selects environments for that role, the resulting binding lets the grantee run DDL/DML **directly in SQL Editor** in those environments **without going through the change-request approval flow**. Today this is severely under-communicated:

1. **Admin grant drawer** (`MembersPage.tsx`) shows muted helper text: *"Allow running DDL/DML statements in the selected environments."* This phrasing is easily misread as "users can submit DDL/DML change requests for these environments" rather than "users bypass change-request approval entirely."
2. **Developer request drawer** (`RequestRoleSheet.tsx`) shows **no copy at all** under the Environments field — the developer doesn't know they're requesting an approval-bypass scope.
3. **Role-grant approval issue page** (`IssueDetailRoleGrantDetails.tsx`) does **not display environments at all** and shows no warning, so approvers cannot see (or be warned about) which environments a request will grant direct DDL/DML access in.
4. **Member-list "current binding" info banner** (`MembersPage.tsx`) styles the same risk as a muted blue *info* banner, which contradicts the warning treatment we are introducing on the other surfaces.

The root cause is that the wording frames the grant around *the role* ("this role allows DDL/DML") when the actual risk is *the environment selection* — selecting environment X turns it into a free-fire zone for that role, bypassing the change-request approval policy. The wording must center on the environment.

## Design

### 1. Risk-model framing

The warning is structured around the same single sentence: in the chosen environment(s), DDL/DML can be run **directly in SQL Editor without approval**. The wording is **permission-aware**: which statement kinds the warning names depends on the perms the role actually has.

| Role grants (env-scoped perms) | Statement-kind label |
|---|---|
| `bb.sql.ddl` + `bb.sql.dml` | `DDL/DML` |
| only `bb.sql.ddl` | `DDL` |
| only `bb.sql.dml` | `DML` |
| neither | (no env field, no warning — existing behavior) |

### 2. New helper: `getRoleEnvironmentLimitationKind`

Replace `roleHasEnvironmentLimitation(role): boolean` (in `frontend/src/components/ProjectMember/utils.ts`) with:

```ts
export type EnvLimitationKind = "DDL" | "DML" | "DDL/DML";

// Returns the human-readable statement-kind label for the role, or undefined
// if the role has no environment-scoped DDL/DML permissions (i.e. no env
// field is shown for this role).
export const getRoleEnvironmentLimitationKind =
  (role: string) => EnvLimitationKind | undefined;
```

Implementation reads the role's permission set (via `useRoleStore`/`checkRoleContainsAnyPermission`) and returns the matching label.

All current call sites of `roleHasEnvironmentLimitation` switch to checking `getRoleEnvironmentLimitationKind(role) !== undefined`. The boolean wrapper is **deleted** — call sites are limited to `MembersPage.tsx`, `RequestRoleSheet.tsx`, and `RoleGrantPanel.tsx` (verified via grep). Any third-party caller will surface as a TS error at build time.

### 3. i18n keys

#### Drawer copy (admin grant + developer request — shared)

```
project.members.ddl-warning:
  "In the selected environments, {{kind}} statements can be directly run in SQL Editor without approval."
```

`{{kind}}` is interpolated as `"DDL"`, `"DML"`, or `"DDL/DML"`. The literal `kind` strings are not translated — they're product permission names.

#### Issue-page copy (approver — dedicated)

```
issue.role-grant.ddl-warning:
  "If approved, in {{environments}}, {{kind}} statements can be directly run in SQL Editor without further approval."
```

`{{environments}}` is interpolated as a comma-joined list of env display titles (e.g. `Prod, Test`).

#### Member-list current-binding banner (rewrites of three existing keys)

```
project.members.ddl-current-some:
  "{{kind}} statements can be directly run in SQL Editor in the listed environments without approval."
project.members.ddl-current-all:
  "{{kind}} statements can be directly run in SQL Editor in ALL environments without approval."
project.members.ddl-current-none:
  "{{kind}} statements are not allowed to be run directly in SQL Editor in any environment."
```

`ALL` is intentionally uppercase in `ddl-current-all` to make the unrestricted scope visually obvious to admins scanning a list of bindings.

#### Removed keys

Delete from `frontend/src/react/locales/en-US.json` and `frontend/src/locales/en-US.json`:

```
project.members.allow-ddl
project.members.allow-ddl-all-environments
project.members.disallow-ddl-all-environments
```

(Vue-locale copies are dead — only React surfaces use the keys, verified via grep.)

Other locale files (`zh-CN.json`, etc.) get the new keys with the English string as a placeholder; translation lands in a follow-up PR.

### 4. UI treatment per surface

#### 4.1 Admin grant drawer (`MembersPage.tsx`, env section)

- Remove the muted helper-text caption under the Environments label.
- Render `<DDLWarningCallout kind={kind} variant="drawer" />` **immediately under the env multi-select**, only when `getRoleEnvironmentLimitationKind(role)` returns a kind.
- Visual: yellow background (`bg-warning-bg`) / yellow border, ⚠ icon, copy from `project.members.ddl-warning`.

#### 4.2 Developer request drawer (`RequestRoleSheet.tsx`, env section)

- Net-add: render the same `<DDLWarningCallout>` under the env multi-select, gated identically.

Both drawers reuse one shared component so styling stays in lockstep.

#### 4.3 Issue page (`IssueDetailRoleGrantDetails.tsx`)

Two changes inside the existing bordered details card:

- **New Environments row**, between Permissions and Database, rendered only when `condition?.environments?.length > 0`. Renders env chips/badges using the same `EnvironmentLabel` component already in use elsewhere.
- **Top warning Alert** (Option A from brainstorming mockups), the first child inside the bordered card. Yellow Alert with ⚠ icon, copy from `issue.role-grant.ddl-warning`.
  Renders only when **both**:
  - `getRoleEnvironmentLimitationKind(issue.roleGrant.role)` returns a kind, **and**
  - `condition?.environments?.length > 0`

Backup placement (Option B from brainstorming): if review feedback during implementation finds the top banner too heavy, the warning can move to a contextual callout immediately under the Environments row. Same copy; same component.

#### 4.4 Member-list current-binding banner (`MembersPage.tsx`)

- Restyle from blue info → yellow warning (`Alert variant="warning"`).
- Use the same `<DDLWarningCallout>` component with `variant="binding-some" | "binding-all" | "binding-none"` so the string switch happens in one place.
- Gating unchanged: only renders when `getRoleEnvironmentLimitationKind(binding.role) !== undefined` (this is the same set of roles that previously triggered `envLimitation`, so visible bindings don't change).

### 5. New shared component: `DDLWarningCallout`

```tsx
// frontend/src/react/components/role-grant/DDLWarningCallout.tsx

type Variant =
  | "drawer"          // admin grant + developer request
  | "issue"           // approver issue page
  | "binding-some"    // member list, specific envs
  | "binding-all"     // member list, no env restriction
  | "binding-none";   // member list, empty envs

interface Props {
  kind: EnvLimitationKind;        // "DDL" | "DML" | "DDL/DML"
  variant: Variant;
  environments?: string[];        // required for "issue"; ignored otherwise
}
```

Internally it picks the right i18n key, interpolates `kind` (and `environments` when relevant), and renders an `Alert` with the standard warning chrome.

### 6. Tests

- **Unit test** for `getRoleEnvironmentLimitationKind`: covers `bb.sql.ddl` only, `bb.sql.dml` only, both, neither, and a built-in role that mixes env-scoped perms with non-env-scoped ones.
- **`MembersPage` test**: switching the selected role updates the kind label in the warning callout (`DDL`, `DML`, `DDL/DML`); for a non-DDL role the env field and warning are both hidden.
- **`RequestRoleSheet.test.tsx`**: extend the existing test file to assert the warning copy renders for a DDL/DML role and is absent for a SELECT-only role.
- **`IssueDetailRoleGrantDetails` test (new)**:
  - Warning + Environments row render for a role-grant issue with `bb.sql.ddl` + `bb.sql.dml` and two envs.
  - Warning hidden when role lacks DDL/DML perms (e.g. SELECT-only role).
  - Environments row hidden when `condition.environments` is empty/undefined.
  - Env-name interpolation: warning text contains the env display titles, not env resource names.
- **`DDLWarningCallout` snapshot/RTL test**: each `variant` × representative `kind` renders the expected i18n string.

### 7. Out of scope

- No backend changes. CEL expression generation, `RoleGrant` proto, and the approval runner are untouched.
- Translation of new keys into non-English locales lands in a follow-up.
- The drawer label *Environments* itself is unchanged.
- No change to which roles support env scoping; we only change how the existing scoping is *communicated*.

## Open questions

None. All ambiguity resolved during brainstorming.

## Rollout / risk

- Pure frontend wording + styling change. No data migration, no breaking-change review needed.
- The shared `DDLWarningCallout` component centralizes copy so future audits or A/B tweaks change one file.
- A small risk of broken tests in the existing `MembersPage` and `RequestRoleSheet` suites because the rendered copy changes — they will be updated as part of the same PR.
