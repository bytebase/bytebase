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
export function getRoleEnvironmentLimitationKind(
  role: string
): EnvLimitationKind | undefined;
```

**Read path.** Implementation is a non-hook utility (call sites include `membersPageEnvironment.ts`, which is not a React component). It reads the role from the Pinia store synchronously via `useRoleStore().getRoleByName(role)` and uses the existing `checkRoleContainsAnyPermission(role, "bb.sql.ddl")` / `("bb.sql.dml")` predicate to test each statement-kind permission separately, then composes the resulting label. This matches how the sibling helper `roleHasDatabaseLimitation` is written today, so the two stay structurally consistent.

**Unknown role.** If the role is not yet loaded into the store (e.g. an issue page that races role-store hydration), `getRoleByName` returns `undefined` (per `frontend/src/store/modules/role.ts:23-25`, which is `roleList.value.find(...)`). The helper short-circuits and returns `undefined` — symmetric with today's `roleHasEnvironmentLimitation` returning `false` for an unknown role. Once the store hydrates, React re-renders and the helper returns the correct label. Document this in a docstring.

**Call-site migration.** All current call sites switch to checking `getRoleEnvironmentLimitationKind(role) !== undefined`. The boolean wrapper is **deleted**. Verified call sites (`grep -rn "roleHasEnvironmentLimitation"` across `frontend/src/`):

| File | Notes |
|------|-------|
| `frontend/src/components/ProjectMember/utils.ts` | Definition. Replace. |
| `frontend/src/react/pages/settings/MembersPage.tsx` | 2 callers (drawer env-field gate, drawer submit-time env scoping). Switch both to the new helper. |
| `frontend/src/react/pages/settings/RequestRoleSheet.tsx` | 1 caller (drawer env-field gate). Switch. |
| `frontend/src/react/pages/settings/membersPageEnvironment.ts` | 1 caller in `getProjectRoleBindingEnvironmentLimitationState`. Switch. |
| `frontend/src/react/pages/settings/RequestRoleSheet.test.tsx` | Mocks the helper. Update to mock `getRoleEnvironmentLimitationKind`. |
| `frontend/src/react/pages/settings/membersPageEnvironment.test.ts` | Mocks the helper. Update mock. |

(There is no production caller in `RoleGrantPanel.tsx` — earlier draft of this spec was wrong. The Vue tree's `frontend/src/components/RoleGrantPanel/` contains only `DatabaseResourceForm/` and `MaxRowCountSelect.vue`; neither references this helper.)

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

The old keys live in **10 locale files** across both the Vue and React i18n trees. Removal scope:

- `frontend/src/react/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove all three keys: `allow-ddl`, `allow-ddl-all-environments`, `disallow-ddl-all-environments`. (React surfaces are the only runtime callers.)
- `frontend/src/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove the single key `allow-ddl` only. (The Vue locale tree contains only `allow-ddl` for these surfaces; the other two keys never existed there. Confirmed by direct inspection of all five files.)

Verified via grep that no `.vue`/`.ts`/`.tsx` source under `frontend/src/` outside of locale files references any of the three old keys other than `MembersPage.tsx` (the surface we're rewriting).

Non-English React locales (`zh-CN.json`, `ja-JP.json`, `es-ES.json`, `vi-VN.json`) get the **new keys** added with translations produced **during implementation in the same PR**. Translation is part of the implementation plan, not deferred — the existing `allow-ddl` is fully translated in all four non-EN locales today, and we should not regress that. The implementation plan will list each (locale, key) pair as an explicit task so translators / the implementer can sweep them together.

### 4. UI treatment per surface

#### 4.1 Admin grant drawer (`MembersPage.tsx`, env section)

- Remove the muted helper-text caption under the Environments label.
- Render `<DDLWarningCallout kind={kind} type="drawer" />` **immediately under the env multi-select**, only when `getRoleEnvironmentLimitationKind(role)` returns a kind.
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

**Edge cases (explicit):**

| Situation | Behavior | Rationale |
|---|---|---|
| Role has DDL/DML perms, `condition.environments` is `undefined` | Warning hidden; Environments row hidden. | Today's request flow always emits an `environments` clause when `getRoleEnvironmentLimitationKind(role) !== undefined`, so undefined is only seen for legacy / hand-edited bindings. Approver still sees role title + permissions. |
| Role has DDL/DML perms, `condition.environments` is `[]` (explicit empty) | Warning hidden; Environments row hidden. | Reachable: the request flow emits `environments: []` when the developer requests a DDL/DML role but selects zero environments. Backend semantics: `resource.environment_id in []` is false everywhere, so the binding grants **no** DDL/DML access — there is no risk to convey, hence no warning. The Environments row is also suppressed because rendering "Environments: (none)" alongside a request that had a DDL/DML-shaped role would invite the misread "this role grants DDL/DML somewhere"; rather than disambiguate via a fourth UI state we let the role's permission list speak for itself. If user research later flags this as confusing, surfacing an explicit "no environments selected" indicator is a follow-up spec. |
| Role-grant issue references a role that has been **deleted** by the time an approver opens the page | `useRoleStore().getRoleByName` returns `undefined` → helper returns `undefined` → warning hides. | Symmetric with today's silent-hide for unknown roles. Approver still sees the existing role-name and database details; they can revoke/decline as usual. |
| Role store hasn't hydrated yet | Helper returns `undefined` on first render, warning hidden; React re-renders once the store loads and warning appears. | Matches today's gating behavior; no flash-of-warning on subsequent loads because the store is cached after first hydration. |
| `kind` would be the empty string | Cannot occur — the warning component is only mounted when `getRoleEnvironmentLimitationKind` returns one of three explicit string literals. **Implementers must NOT add a defensive `kind ?? "DDL/DML"` fallback** — that would mask a logic bug elsewhere. |

Placement is **Option A only** for this plan. Option B (contextual callout under the Environments row) was discussed during brainstorming as a fallback but is intentionally **not** included as a mid-implementation pivot — if review pushes back on Option A, we treat that as a separate spec amendment rather than letting the plan absorb both.

#### 4.4 Member-list current-binding banner (`MembersPage.tsx`)

- Restyle from blue info → yellow warning (`Alert variant="warning"`).
- Use the same `<DDLWarningCallout>` component with one of three variants so the string switch happens in one place.
- Gating unchanged: only renders when `getRoleEnvironmentLimitationKind(binding.role) !== undefined` (this is the same set of roles that previously triggered `envLimitation`, so visible bindings don't change).

**Variant ↔ existing limitation-state mapping.** `getProjectRoleBindingEnvironmentLimitationState` (in `membersPageEnvironment.ts`) already returns the three states we need:

| Existing limitation state | New `<DDLWarningCallout>` variant | Visual rendering |
|---|---|---|
| `{ type: "unrestricted" }` | `binding-all` | Warning Alert, copy from `ddl-current-all`. **No** env badges. |
| `{ type: "restricted", environments: [...] }` (length > 0) | `binding-some` | Warning Alert, copy from `ddl-current-some`, **plus** env badges rendered next to or under the copy (mirrors the existing layout). |
| `{ type: "restricted", environments: [] }` | `binding-none` | Warning Alert, copy from `ddl-current-none`. No env badges. |
| `undefined` | (don't render) | (banner hidden — no DDL/DML scope on this role.) |

The component receives the limitation state directly (or the variant + optional env list) — implementer's call. The mapping table above is the source of truth.

### 5. New shared component: `DDLWarningCallout`

```tsx
// frontend/src/react/components/role-grant/DDLWarningCallout.tsx

type DDLWarningProps =
  | { type: "drawer";       kind: EnvLimitationKind }
  | { type: "issue";        kind: EnvLimitationKind; environments: string[] }
  | { type: "binding-some"; kind: EnvLimitationKind }
  | { type: "binding-all";  kind: EnvLimitationKind }
  | { type: "binding-none"; kind: EnvLimitationKind };
```

The discriminated union enforces at compile time that callers provide `environments` exactly when the variant needs them — no `environments?` footgun where a caller forgets to pass them and silently renders an empty list.

Internally:
- Picks the right i18n key per variant.
- Interpolates `kind` always; interpolates `environments` (comma-joined display titles) only for `issue`.
- Renders an `Alert` with `variant="warning"` chrome (yellow background, ⚠ icon).

The component does **not** render env badges itself. Consumers that need badges (issue page's Environments row, member-list `binding-some` banner) render them outside the callout because each surface arranges them differently relative to surrounding rows. Centralizing badge layout in the component would couple it to one surface's layout decisions; the centralized thing here is **copy + i18n key routing**.

### 6. Tests

- **Unit test** for `getRoleEnvironmentLimitationKind`: covers `bb.sql.ddl` only, `bb.sql.dml` only, both, neither, a built-in role that mixes env-scoped with non-env-scoped perms, and the **unknown-role** case (returns `undefined` to lock the symmetry-with-old-`false` behavior).
- **`MembersPage` test**: switching the selected role updates the kind label in the warning callout (`DDL`, `DML`, `DDL/DML`); for a non-DDL role the env field and warning are both hidden. Member-list banner renders the correct variant for each of `unrestricted` / `restricted` (non-empty) / `restricted` (empty).
- **`RequestRoleSheet.test.tsx`**: existing mock of `roleHasEnvironmentLimitation` switches to mocking `getRoleEnvironmentLimitationKind`. Add assertions: the warning copy renders for a DDL/DML role and is absent for a SELECT-only role.
- **`membersPageEnvironment.test.ts`**: existing mock switches to `getRoleEnvironmentLimitationKind`. Existing tri-state assertions stay green; add a case for the new mock returning `undefined` (still produces `undefined` from `getProjectRoleBindingEnvironmentLimitationState`).
- **`IssueDetailRoleGrantDetails` test (new)**:
  - Warning + Environments row render for a role-grant issue with `bb.sql.ddl` + `bb.sql.dml` and two envs.
  - Warning hidden when role lacks DDL/DML perms (e.g. SELECT-only role).
  - Environments row hidden when `condition.environments` is empty/undefined.
  - Env-name interpolation: warning text contains the env display titles, not env resource names.
  - Role-deleted case: helper returns `undefined`, warning is hidden, rest of details still render.
- **`DDLWarningCallout` snapshot/RTL test**: each variant × representative `kind` renders the expected i18n string. Type-level test (compile-only) confirms the discriminated union rejects passing `environments` to `binding-all` and rejects omitting `environments` from `issue`.

### 7. Out of scope

- No backend changes. CEL expression generation, `RoleGrant` proto, and the approval runner are untouched.
- The drawer label *Environments* itself is unchanged.
- No change to which roles support env scoping; we only change how the existing scoping is *communicated*.
- Surfacing a warning when a role has DDL/DML perms but `condition.environments` is empty/undefined (today this combination is unreachable from the request flow — see §4.3 edge cases).

## Known limitations (accepted)

- **`binding-all` copy ("ALL environments") describes env scope only.** A binding can be `unrestricted` on environments while still being database-scoped (DDL/DML allowed in selected databases × all environments). The warning conveys the env axis only — not the database axis. This is consistent with the warning being attached to environment selection. If user research later flags this as misleading, surfacing the database scope alongside is a follow-up spec; the current scope is intentionally narrow.

- **Async layout shift on the issue page.** `condition.environments` is parsed inside an awaited `convertFromCELString` call in `IssueDetailRoleGrantDetails.tsx`. The Environments row + warning banner therefore appear after first paint, causing a small layout shift at the top of the details card. This already happens today for the database row; the new rows continue the existing pattern. If this becomes noticeable in user research, a skeleton placeholder can land in a follow-up.

- **No equivalent surface in the SQL Editor self-service flow.** Verified via grep: `frontend/src/react/components/sql-editor/RoleGrantPanel.tsx` does not use `EnvironmentMultiSelect` — it grants per-database, not per-environment. The warning is irrelevant there. The Linear ticket title's "SQL Editor self-service" refers to the developer request flow (`RequestRoleSheet`), which is covered.

## Open questions

None. All ambiguity resolved during brainstorming.

## Rollout / risk

- Pure frontend wording + styling change. No data migration, no breaking-change review needed.
- The shared `DDLWarningCallout` component centralizes copy so future audits or A/B tweaks change one file.
- A small risk of broken tests in the existing `MembersPage` and `RequestRoleSheet` suites because the rendered copy changes — they will be updated as part of the same PR.
