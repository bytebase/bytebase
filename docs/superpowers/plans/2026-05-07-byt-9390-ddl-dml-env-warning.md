# BYT-9390 DDL/DML Environment-Grant Warnings — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current muted/blue messaging on four role-grant surfaces with a permission-aware yellow warning that frames the risk as "selecting an environment lets the grantee run DDL/DML directly in SQL Editor without approval."

**Architecture:** A new permission-aware helper (`getRoleEnvironmentLimitationKind`) replaces the boolean `roleHasEnvironmentLimitation`. A new shared component (`DDLWarningCallout`) renders the warning copy in five surface variants (admin grant drawer, developer request drawer, approver issue page, member-list banner with envs, member-list banner without envs). **Five** new i18n keys (4 under `project.members.*`, 1 under `issue.role-grant.*`) replace **three** old keys across 10 locale files (Vue tree + React tree × 5 languages).

**Semantic-equivalence note (anchor):** `getRoleEnvironmentLimitationKind` returns a defined value iff the role has `bb.sql.ddl` and/or `bb.sql.dml`. The boolean it replaces, `roleHasEnvironmentLimitation`, also checks exactly that pair (verified: `checkRoleContainsAnyPermission(role, "bb.sql.ddl", "bb.sql.dml")`). So `getRoleEnvironmentLimitationKind(role) !== undefined` is **bit-identical** to the old gate — no role's banner-visibility changes. Do not confuse with the *sibling* `roleHasDatabaseLimitation`, which checks five permissions including `bb.sql.select`; that helper is unchanged.

**Tech Stack:** React, TypeScript, Tailwind v4 + Base UI, vitest + React Testing Library, vue-i18n locale files (JSON).

**Spec:** `docs/superpowers/specs/2026-05-07-byt-9390-ddl-dml-env-warning-design.md`

---

## File Structure

**Created files:**
- `frontend/src/react/components/role-grant/DDLWarningCallout.tsx` — shared warning component (discriminated-union props).
- `frontend/src/react/components/role-grant/DDLWarningCallout.test.tsx` — RTL tests for each variant.
- `frontend/src/components/ProjectMember/utils.test.ts` — vitest unit test for `getRoleEnvironmentLimitationKind`.
- `frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.test.tsx` — new test covering Environments row + warning banner.

**Modified files:**
- `frontend/src/components/ProjectMember/utils.ts` — replace boolean helper.
- `frontend/src/react/pages/settings/MembersPage.tsx` — switch helper; replace drawer helper-text + member-list banner.
- `frontend/src/react/pages/settings/RequestRoleSheet.tsx` — switch helper; add warning under env multiselect.
- `frontend/src/react/pages/settings/membersPageEnvironment.ts` — switch helper.
- `frontend/src/react/pages/settings/RequestRoleSheet.test.tsx` — update mock; add assertions.
- `frontend/src/react/pages/settings/membersPageEnvironment.test.ts` — update mock; add `undefined` case.
- `frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx` — add Environments row + warning banner.
- `frontend/src/react/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove 3 old keys, add 5 new keys.
- `frontend/src/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove 1 old key (`allow-ddl`).

**Why this layout:** The new component lives under `react/components/role-grant/` (a new folder) because the same warning is consumed from three call-site domains (settings page, request drawer, issue-detail page). Putting it in any one of those domain folders would couple it to that domain. The helper change stays in `components/ProjectMember/utils.ts` so it remains discoverable next to its sibling `roleHasDatabaseLimitation`.

---

## Chunk 1: Foundation — helper, new EN copy, shared component

### Task 1: Replace `roleHasEnvironmentLimitation` with `getRoleEnvironmentLimitationKind`

**Files:**
- Modify: `frontend/src/components/ProjectMember/utils.ts`
- Create: `frontend/src/components/ProjectMember/utils.test.ts`

- [ ] **Step 1.1: Write the failing unit test.**

```ts
// frontend/src/components/ProjectMember/utils.test.ts
import { describe, expect, test, vi } from "vitest";
import { getRoleEnvironmentLimitationKind } from "./utils";

// Stub the role store — getRoleEnvironmentLimitationKind reads it once
// per call. Other @/utils + @/store exports are stubbed as no-ops so a
// sibling export's transitive imports don't crash.
const fixtures: Record<string, string[]> = {
  "roles/sqlEditorUser":     ["bb.sql.ddl", "bb.sql.dml"],
  "roles/sqlEditorDDLOnly":  ["bb.sql.ddl"],
  "roles/sqlEditorDMLOnly":  ["bb.sql.dml"],
  "roles/queryOnly":         ["bb.sql.select"],
  "roles/projectViewer":     [],
};
vi.mock("@/store", () => ({
  useRoleStore: () => ({
    getRoleByName: (role: string) =>
      fixtures[role] === undefined
        ? undefined
        : { name: role, permissions: fixtures[role] },
  }),
}));
vi.mock("@/utils", () => ({
  displayRoleTitle: (r: string) => r,
  checkRoleContainsAnyPermission: () => false, // unused by the new helper
}));

describe("getRoleEnvironmentLimitationKind", () => {
  test("returns 'DDL/DML' when role has both ddl and dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorUser")).toBe(
      "DDL/DML"
    );
  });

  test("returns 'DDL' when role has only ddl", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorDDLOnly")).toBe(
      "DDL"
    );
  });

  test("returns 'DML' when role has only dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorDMLOnly")).toBe(
      "DML"
    );
  });

  test("returns undefined when role has neither ddl nor dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/queryOnly")).toBeUndefined();
  });

  test("returns undefined for an unknown role", () => {
    expect(
      getRoleEnvironmentLimitationKind("roles/doesNotExist")
    ).toBeUndefined();
  });
});
```

- [ ] **Step 1.2: Run the test and confirm it fails.**

Run: `pnpm --dir frontend test -- src/components/ProjectMember/utils.test.ts`
Expected: FAIL — `getRoleEnvironmentLimitationKind is not a function` (or "no exported member").

- [ ] **Step 1.3: Implement the helper.**

Replace the body of `frontend/src/components/ProjectMember/utils.ts`:

```ts
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { useRoleStore } from "@/store";
import { checkRoleContainsAnyPermission, displayRoleTitle } from "@/utils";

export const getBindingIdentifier = (binding: Binding): string => {
  const identifier = [displayRoleTitle(binding.role)];
  if (binding.condition && binding.condition.expression) {
    identifier.push(binding.condition.expression);
  }
  return identifier.join(".");
};

export const roleHasDatabaseLimitation = (role: string) => {
  return checkRoleContainsAnyPermission(
    role,
    "bb.sql.select",
    "bb.sql.ddl",
    "bb.sql.dml",
    "bb.sql.explain",
    "bb.sql.info"
  );
};

// {{kind}} is spliced raw into translated strings — do not localize.
export type EnvLimitationKind = "DDL" | "DML" | "DDL/DML";

// undefined ⇔ role has no env-scoped permissions ⇔ caller hides the env section.
// Reads the role once (vs. two checkRoleContainsAnyPermission calls) so the
// hot path on member-list / drawer renders touches the role store once.
export const getRoleEnvironmentLimitationKind = (
  role: string
): EnvLimitationKind | undefined => {
  const r = useRoleStore().getRoleByName(role);
  if (!r) return undefined;
  const perms = new Set(r.permissions);
  const hasDDL = perms.has("bb.sql.ddl");
  const hasDML = perms.has("bb.sql.dml");
  if (hasDDL && hasDML) return "DDL/DML";
  if (hasDDL) return "DDL";
  if (hasDML) return "DML";
  return undefined;
};

// Transitional shim — keeps the build green while call sites migrate.
// Removed in Task 10 once the last caller is gone.
// @deprecated use getRoleEnvironmentLimitationKind instead.
export const roleHasEnvironmentLimitation = (role: string): boolean =>
  getRoleEnvironmentLimitationKind(role) !== undefined;
```

Note on the `useRoleStore()` call: this is a Pinia store accessor (returns a cached singleton), not a React hook. It's safe to call from a non-component utility — that's the same pattern `IssueDetailRoleGrantDetails.tsx` already uses inline.

The boolean `roleHasEnvironmentLimitation` is kept as a **transitional shim** that delegates to the new helper. Tasks 4–7 migrate each production call site to `getRoleEnvironmentLimitationKind`; Task 10 deletes the shim once the last caller is gone. This keeps `git bisect` and partial-checkout workflows working across the migration.

- [ ] **Step 1.4: Run the test and confirm it passes.**

Run: `pnpm --dir frontend test -- src/components/ProjectMember/utils.test.ts`
Expected: PASS — 5 cases.

- [ ] **Step 1.5: Run full type-check to confirm tree still builds.**

Run: `pnpm --dir frontend type-check`
Expected: PASS. The transitional shim keeps existing call sites compiling. Tasks 4–7 migrate each call site one at a time; the shim is removed in Task 10.

- [ ] **Step 1.6: Commit.**

```bash
git add frontend/src/components/ProjectMember/utils.ts \
        frontend/src/components/ProjectMember/utils.test.ts
git commit -m "feat(role-grant): add getRoleEnvironmentLimitationKind helper

Adds a permission-aware helper (returns 'DDL' / 'DML' / 'DDL/DML' /
undefined) that will power upcoming DDL/DML warnings. The boolean
roleHasEnvironmentLimitation stays as a transitional shim until call
sites migrate; final shim removal in the cleanup commit.

Refs: BYT-9390"
```

---

### Task 2: Add new English i18n keys (both Vue + React trees)

**Files:**
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/locales/en-US.json` (no new keys here — Vue tree only loses `allow-ddl` later, no additions needed)

We add only the **React tree** EN keys here. (The Vue tree never sees these surfaces.) Non-EN React locales pick up the new keys in Task 9; old-key removal is Task 10.

- [ ] **Step 2.1: Add the five new keys to `frontend/src/react/locales/en-US.json`.**

Insert under `project.members.*` (alphabetical placement near the existing `allow-ddl*` entries):

```json
"ddl-current-all": "{{kind}} statements can be directly run in SQL Editor in ALL environments without approval.",
"ddl-current-none": "{{kind}} statements are not allowed to be run directly in SQL Editor in any environment.",
"ddl-current-some": "{{kind}} statements can be directly run in SQL Editor in the listed environments without approval.",
"ddl-warning": "In the selected environments, {{kind}} statements can be directly run in SQL Editor without approval.",
```

Insert under `issue.role-grant.*` (next to existing role-grant keys):

```json
"ddl-warning": "If approved, in {{environments}}, {{kind}} statements can be directly run in SQL Editor without further approval.",
```

Do **not** remove the old keys yet — they stay live until Task 10's sweep. Existing call sites still reference them.

- [ ] **Step 2.2: Lint the JSON.**

Run: `pnpm --dir frontend fix`
Expected: no errors. Biome will sort keys alphabetically inside each object — this is the canonical layout.

- [ ] **Step 2.3: Commit.**

```bash
git add frontend/src/react/locales/en-US.json
git commit -m "i18n(en): add new ddl-warning + ddl-current-* keys

New copy frames the risk around environment selection and is
permission-aware ({{kind}} = DDL | DML | DDL/DML). Surfaces consume
these in subsequent commits; old keys removed in the final sweep.

Refs: BYT-9390"
```

---

### Task 3: Create the shared `DDLWarningCallout` component

**Files:**
- Create: `frontend/src/react/components/role-grant/DDLWarningCallout.tsx`
- Create: `frontend/src/react/components/role-grant/DDLWarningCallout.test.tsx`

- [ ] **Step 3.1: Write the failing component tests.**

```tsx
// frontend/src/react/components/role-grant/DDLWarningCallout.test.tsx
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { DDLWarningCallout } from "./DDLWarningCallout";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    // Return the key + a JSON-encoded vars suffix so assertions can match
    // both the i18n key and the interpolated values. We don't have access
    // to the locale files in unit tests, so the actual locale value isn't
    // rendered — just the key + vars, which is sufficient to verify the
    // component picked the right key and threaded the right interpolation.
    t: (key: string, vars?: Record<string, unknown>) => {
      const parts = [key];
      if (vars) parts.push(JSON.stringify(vars));
      return parts.join(" ");
    },
  }),
}));

describe("DDLWarningCallout", () => {
  test("drawer variant renders ddl-warning copy with kind interpolated", () => {
    render(<DDLWarningCallout type="drawer" kind="DDL/DML" />);
    expect(
      screen.getByText(/project.members.ddl-warning/),
    ).toHaveTextContent("DDL/DML");
  });

  test("issue variant renders issue.role-grant.ddl-warning with environments", () => {
    render(
      <DDLWarningCallout
        type="issue"
        kind="DDL"
        environments={["Prod", "Test"]}
      />,
    );
    const node = screen.getByText(/issue.role-grant.ddl-warning/);
    expect(node).toHaveTextContent("DDL");
    expect(node).toHaveTextContent("Prod, Test");
  });

  test("binding-some renders ddl-current-some copy", () => {
    render(
      <DDLWarningCallout
        type="binding-some"
        kind="DML"
      />,
    );
    expect(
      screen.getByText(/project.members.ddl-current-some/),
    ).toHaveTextContent("DML");
    // Pin the contract: the warning copy itself must NOT interpolate
    // env names. Consumers (e.g. MembersPage) render env badges as a
    // separate row next to the warning. If a future implementer adds
    // env interpolation to the i18n string, this assertion catches it.
    expect(screen.queryByText(/Staging/)).not.toBeInTheDocument();
  });

  test("binding-all renders ddl-current-all copy", () => {
    render(<DDLWarningCallout type="binding-all" kind="DDL/DML" />);
    expect(
      screen.getByText(/project.members.ddl-current-all/),
    ).toHaveTextContent("DDL/DML");
  });

  test("binding-none renders ddl-current-none copy", () => {
    render(<DDLWarningCallout type="binding-none" kind="DML" />);
    expect(
      screen.getByText(/project.members.ddl-current-none/),
    ).toHaveTextContent("DML");
  });

  test("renders an alert with role=alert", () => {
    render(<DDLWarningCallout type="drawer" kind="DDL" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

});

// Type-level assertions — these functions are NEVER called. They live
// outside any `it()` block so vitest doesn't execute them. Their sole
// purpose is to fail `pnpm --dir frontend type-check` if the union
// shape ever stops rejecting invalid prop combinations. (Spec §6.)
//
// `void _typeChecks` keeps eslint from flagging the function as unused.
const _typeChecks = () => {
  // @ts-expect-error — `binding-all` does not accept `environments`.
  const _a = <DDLWarningCallout type="binding-all" kind="DDL" environments={["x"]} />;
  // @ts-expect-error — `issue` requires `environments`.
  const _b = <DDLWarningCallout type="issue" kind="DDL" />;
  // @ts-expect-error — `drawer` does not accept `environments`.
  const _c = <DDLWarningCallout type="drawer" kind="DDL" environments={["x"]} />;
  return [_a, _b, _c];
};
void _typeChecks;
```

The thunk is never invoked, so `render(...)` calls never trip on missing `environments` at runtime — but `tsc` still type-checks the JSX inside, which is where the `@ts-expect-error` assertions fire.

- [ ] **Step 3.2: Run the tests and confirm they fail.**

Run: `pnpm --dir frontend test -- src/react/components/role-grant/DDLWarningCallout.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3.3: Implement the component.**

Verified precondition: `frontend/src/react/components/ui/alert.tsx` already exports `Alert` with `variant="warning"` (yellow background, ⚠ icon via `AlertTriangle`). No alert-primitive work needed.

```tsx
// frontend/src/react/components/role-grant/DDLWarningCallout.tsx
import { useTranslation } from "react-i18next";
import type { EnvLimitationKind } from "@/components/ProjectMember/utils";
import { Alert } from "@/react/components/ui/alert";

// Discriminated union (matches the codebase's `type:` discriminator
// convention — see usePagedData.tsx, useDropdown.ts).
// `environments` is required exactly when the variant interpolates
// them ("issue") and absent everywhere else. Consumers needing to
// render env badges (e.g. binding-some) carry the env list themselves
// and render it as a separate row.
type DDLWarningProps =
  | { type: "drawer"; kind: EnvLimitationKind }
  | { type: "issue"; kind: EnvLimitationKind; environments: string[] }
  | { type: "binding-some"; kind: EnvLimitationKind }
  | { type: "binding-all"; kind: EnvLimitationKind }
  | { type: "binding-none"; kind: EnvLimitationKind };

const typeToKey: Record<DDLWarningProps["type"], string> = {
  drawer: "project.members.ddl-warning",
  issue: "issue.role-grant.ddl-warning",
  "binding-some": "project.members.ddl-current-some",
  "binding-all": "project.members.ddl-current-all",
  "binding-none": "project.members.ddl-current-none",
};

export function DDLWarningCallout(props: DDLWarningProps) {
  const { t } = useTranslation();
  const key = typeToKey[props.type];
  const interpolated =
    props.type === "issue"
      ? t(key, { kind: props.kind, environments: props.environments.join(", ") })
      : t(key, { kind: props.kind });
  return <Alert variant="warning">{interpolated}</Alert>;
}
```

Notes:
- `binding-some` does **not** carry `environments` on its prop. The consumer (`MembersPage.tsx`) already has the env list in scope and renders env badges as a separate row next to the warning — keeping the prop union narrow prevents future implementers from accidentally interpolating env names into the warning string.
- Discriminator is named `type` (not `variant`) to match the existing pattern in `frontend/src/react/hooks/usePagedData.tsx:36-45` and `frontend/src/react/components/sql-editor/useDropdown.ts`.
- Bare `Alert` (no title/description) gives a single-line warning, matching the brainstorming mockups.

- [ ] **Step 3.4: Run the tests and confirm they pass.**

Run: `pnpm --dir frontend test -- src/react/components/role-grant/DDLWarningCallout.test.tsx`
Expected: PASS — 6 cases.

- [ ] **Step 3.5: Type-check.**

Run: `pnpm --dir frontend type-check`
Expected: type-check still passes (the shim from Task 1 keeps the tree green) — no new errors introduced by this component.

- [ ] **Step 3.6: Commit.**

```bash
git add frontend/src/react/components/role-grant/
git commit -m "feat(role-grant): add DDLWarningCallout shared component

Single warning component used by admin grant drawer, developer request
drawer, role-grant issue page, and member-list current-binding banner.
Discriminated-union props prevent the env-list footgun.

Refs: BYT-9390"
```

---

## Chunk 2: Migrate drawer call sites

### Task 4: Switch `RequestRoleSheet` to the new helper + add warning

**Files:**
- Modify: `frontend/src/react/pages/settings/RequestRoleSheet.tsx`
- Modify: `frontend/src/react/pages/settings/RequestRoleSheet.test.tsx`

- [ ] **Step 4.1: Update the test mock first (so the failing assertion is meaningful).**

The existing mock at `RequestRoleSheet.test.tsx:157-162` is:

```ts
vi.mock("@/components/ProjectMember/utils", () => ({
  // PROJECT_OWNER is not a SQL-permission role — these return false, so the
  // database/environment scope sections stay hidden in the tests.
  roleHasDatabaseLimitation: () => false,
  roleHasEnvironmentLimitation: () => false,
}));
```

Replace with:

```ts
vi.mock("@/components/ProjectMember/utils", () => ({
  // Wrap in vi.fn() so per-test overrides (vi.mocked(x).mockReturnValue(...))
  // work — plain arrow functions can't be re-mocked at runtime.
  // Default behavior matches the existing test assumption: scope sections
  // stay hidden because PROJECT_OWNER isn't a SQL-permission role.
  roleHasDatabaseLimitation: vi.fn(() => false),
  getRoleEnvironmentLimitationKind: vi.fn(() => undefined),
}));
```

The new test cases in Step 4.2 override the mock per-test using `vi.mocked(...).mockReturnValue(...)`.

- [ ] **Step 4.2: Add a failing assertion for the new warning.**

The existing tests in this file build the sheet via direct `createRoot` + `act` (no `renderSheet` helper exported). Add the new cases at the end of the existing `describe` block, copying the closest existing test's setup verbatim and only adjusting the role + the assertion. Pattern:

```ts
it("renders DDL/DML warning under env multiselect for a sql-editor role", async () => {
  // Override the default helper mock for this case so the env section appears.
  const utilsMock = await import("@/components/ProjectMember/utils");
  vi.mocked(utilsMock.getRoleEnvironmentLimitationKind).mockReturnValue(
    "DDL/DML",
  );

  // ...same arrange/act as the nearest existing test, with role set to
  // "roles/sqlEditorUser".

  expect(
    container.textContent,
  ).toContain("project.members.ddl-warning");
});

it("does not render the warning for a non-DDL role", async () => {
  // Default mock returns undefined for env kind — no override needed.
  // ...same arrange/act with a SELECT-only role.

  expect(
    container.textContent,
  ).not.toContain("project.members.ddl-warning");
});
```

Use `container.textContent` rather than `screen.getByText` because the existing tests in this file mount with `createRoot` directly and do not register the RTL `screen` helper; copy the existing tests' pattern verbatim. If the existing tests do use `screen`, switch to `screen.queryByText`.

- [ ] **Step 4.3: Run the test and confirm it fails.**

Run: `pnpm --dir frontend test -- src/react/pages/settings/RequestRoleSheet.test.tsx`
Expected: FAIL — warning not rendered.

- [ ] **Step 4.4: Update `RequestRoleSheet.tsx`.**

Existing import block at `RequestRoleSheet.tsx:6-9`:

```ts
import {
  roleHasDatabaseLimitation,
  roleHasEnvironmentLimitation,
} from "@/components/ProjectMember/utils";
```

Replace with:

```ts
import {
  getRoleEnvironmentLimitationKind,
  roleHasDatabaseLimitation,
} from "@/components/ProjectMember/utils";
```

Add a new import block (alphabetical placement among other `@/react/components/...` imports):

```ts
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
```

Existing derivation at `RequestRoleSheet.tsx:251`:

```ts
const showEnvironments = !!role && roleHasEnvironmentLimitation(role);
```

Replace with (`envKind` is its own gate — no separate boolean):

```ts
const envKind = role ? getRoleEnvironmentLimitationKind(role) : undefined;
```

Existing JSX block at `RequestRoleSheet.tsx:536-546`:

```tsx
{showEnvironments && (
  <div className="flex flex-col gap-y-1">
    <label className="text-sm font-medium">
      {t("common.environments")}
    </label>
    <EnvironmentMultiSelect
      value={environments}
      onChange={setEnvironments}
    />
  </div>
)}
```

Replace with (note `gap-y-2` so the warning sits cleanly under the multiselect, and the new callout):

```tsx
{envKind && (
  <div className="flex flex-col gap-y-2">
    <label className="text-sm font-medium">
      {t("common.environments")}
    </label>
    <EnvironmentMultiSelect
      value={environments}
      onChange={setEnvironments}
    />
    <DDLWarningCallout type="drawer" kind={envKind} />
  </div>
)}
```

Gating on `envKind` directly narrows it to `EnvLimitationKind` inside the branch — no separate boolean, no non-null assertion.

- [ ] **Step 4.5: Run the test and confirm it passes.**

Run: `pnpm --dir frontend test -- src/react/pages/settings/RequestRoleSheet.test.tsx`
Expected: PASS — all existing + 2 new cases.

- [ ] **Step 4.6: Commit.**

```bash
git add frontend/src/react/pages/settings/RequestRoleSheet.tsx \
        frontend/src/react/pages/settings/RequestRoleSheet.test.tsx
git commit -m "feat(role-grant): warn on env selection in request drawer

Developers requesting a role with bb.sql.ddl/bb.sql.dml now see a
permission-aware warning under the environment multiselect explaining
that selected envs allow direct DDL/DML execution in SQL Editor without
approval.

Refs: BYT-9390"
```

---

### Task 5: Switch `membersPageEnvironment` (and its test) to the new helper

**Files:**
- Modify: `frontend/src/react/pages/settings/membersPageEnvironment.ts`
- Modify: `frontend/src/react/pages/settings/membersPageEnvironment.test.ts`

- [ ] **Step 5.1: Update the test mock + add the new case (failing first).**

In `membersPageEnvironment.test.ts`, replace the mock:

```ts
vi.mock("@/components/ProjectMember/utils", () => ({
  roleHasEnvironmentLimitation: (role: string) =>
    role === "roles/sqlEditorUser",
}));
```

With:

```ts
vi.mock("@/components/ProjectMember/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));
```

Add a new test case asserting `undefined` propagation:

```ts
test("returns undefined when getRoleEnvironmentLimitationKind returns undefined", () => {
  expect(
    getProjectRoleBindingEnvironmentLimitationState(binding("roles/queryOnly")),
  ).toBeUndefined();
});
```

(Verify whether an equivalent case already exists; if so just adjust the existing assertion's role name.)

- [ ] **Step 5.2: Run the test and confirm it fails.**

Run: `pnpm --dir frontend test -- src/react/pages/settings/membersPageEnvironment.test.ts`
Expected: FAIL — production file still imports `roleHasEnvironmentLimitation`, mock won't match.

- [ ] **Step 5.3: Update the production module.**

In `membersPageEnvironment.ts`:

```ts
import { getRoleEnvironmentLimitationKind } from "@/components/ProjectMember/utils";
// ...
export const getProjectRoleBindingEnvironmentLimitationState = (
  binding: Binding
): ProjectRoleBindingEnvironmentLimitationState | undefined => {
  if (getRoleEnvironmentLimitationKind(binding.role) === undefined) {
    return undefined;
  }
  // ...rest unchanged
};
```

**Semantic equivalence (load-bearing for Task 7).** The new gate `getRoleEnvironmentLimitationKind(role) === undefined` is bit-identical to the old gate `!roleHasEnvironmentLimitation(role)`: both reduce to "role lacks both `bb.sql.ddl` and `bb.sql.dml`." Verify by reading both helpers' definitions in `frontend/src/components/ProjectMember/utils.ts` — they call `checkRoleContainsAnyPermission(role, "bb.sql.ddl", "bb.sql.dml")` against the same permission pair. (Do NOT confuse with the sibling `roleHasDatabaseLimitation`, which checks 5 permissions including `bb.sql.select` — that helper is unchanged.) Therefore the visible-binding set produced by `getProjectRoleBindingEnvironmentLimitationState` does not change for any role. This invariant is what lets Task 7 use `bindingKind` non-null inside the `envLimitation && bindingKind` guard.

The exported `ProjectRoleBindingEnvironmentLimitationState` shape stays the same — `MembersPage.tsx` still consumes it as `unrestricted` / `restricted`. Variant-to-state mapping happens at the rendering layer (Task 7).

- [ ] **Step 5.4: Run the test and confirm it passes.**

Run: `pnpm --dir frontend test -- src/react/pages/settings/membersPageEnvironment.test.ts`
Expected: PASS.

- [ ] **Step 5.5: Commit.**

```bash
git add frontend/src/react/pages/settings/membersPageEnvironment.ts \
        frontend/src/react/pages/settings/membersPageEnvironment.test.ts
git commit -m "refactor(role-grant): switch membersPageEnvironment to kind helper

No behavior change — the tri-state (unrestricted / restricted with
envs / undefined) is preserved. Just swaps the gating predicate.

Refs: BYT-9390"
```

---

### Task 6: Switch `MembersPage` drawer to the new helper + warning

**Files:**
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx` — three regions:
  - **Imports** (top of file): swap the helper import.
  - **Drawer form component** (~L800-940): swap the env-limitation derivation to `envKind`, replace the helper-text caption with `<DDLWarningCallout>`.
  - **Drawer submit handler** (~L1162): swap the second helper call to use `envKind` for the env-scope filter.

  All three regions live inside the drawer form's component file. Task 7 (member-list banner) edits a different region (~L1417-1455) of the same file but does not conflict with these regions. Do **not** change anything outside these line spans.

- [ ] **Step 6.1: Update both `roleHasEnvironmentLimitation` call sites.**

Find at ~L804:

```ts
() => form.role && roleHasEnvironmentLimitation(form.role),
```

Find at ~L1162:

```ts
const environments = roleHasEnvironmentLimitation(form.role)
  ? form.environments
  : undefined;
```

Replace both with calls that thread an `envKind` value through the form-section component. Concretely:

In the form component's body, near the existing memos/derived values, derive once:

```ts
const envKind = form.role
  ? getRoleEnvironmentLimitationKind(form.role)
  : undefined;
```

Then:

```ts
const environments = envKind !== undefined ? form.environments : undefined;
```

Update the import line:

```ts
import {
  getRoleEnvironmentLimitationKind,
  roleHasDatabaseLimitation,
} from "@/components/ProjectMember/utils";
```

(remove `roleHasEnvironmentLimitation` from the import).

- [ ] **Step 6.2: Replace the helper-text caption with the warning component.**

Verify there is exactly one occurrence of this JSX block in the file before editing (`grep -c "project.members.allow-ddl\"" frontend/src/react/pages/settings/MembersPage.tsx` should report 2 — one for the drawer caption and one for the member-list banner that Task 7 handles separately). If the count is higher, read the file and apply each replacement individually with enough surrounding context to disambiguate.

Find the JSX block at ~L918–934:

```tsx
{showEnvironments && (
  <div className="flex flex-col gap-y-2">
    <div>
      <label className="block text-sm font-medium text-control">
        {t("common.environments")}
      </label>
      <span className="text-xs text-control-light">
        {t("project.members.allow-ddl")}
      </span>
    </div>
    <EnvironmentMultiSelect
      value={form.environments}
      onChange={(envs) => onChange({ ...form, environments: envs })}
    />
  </div>
)}
```

Replace with:

```tsx
{envKind && (
  <div className="flex flex-col gap-y-2">
    <label className="block text-sm font-medium text-control">
      {t("common.environments")}
    </label>
    <EnvironmentMultiSelect
      value={form.environments}
      onChange={(envs) => onChange({ ...form, environments: envs })}
    />
    <DDLWarningCallout type="drawer" kind={envKind} />
  </div>
)}
```

Gating on `envKind` directly narrows it to `EnvLimitationKind` inside the branch — no separate boolean, no non-null assertion.

Add the import:

```ts
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
```

The faint helper-text `<span>` is **deleted** (its key `project.members.allow-ddl` will be removed in Task 10).

- [ ] **Step 6.3: Type-check the file.**

Run: `pnpm --dir frontend type-check`
Expected: type-check passes for `MembersPage.tsx` and all files migrated so far. Remaining errors (if any) only in the member-list banner (Task 7) and issue page (Task 8).

- [ ] **Step 6.4: Commit.**

```bash
git add frontend/src/react/pages/settings/MembersPage.tsx
git commit -m "feat(role-grant): warn on env selection in admin grant drawer

Replaces the muted helper-text caption with a permission-aware yellow
warning component under the env multiselect.

Refs: BYT-9390"
```

---

### Task 7: Restyle `MembersPage` member-list current-binding banner

**Files:**
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx` (banner block, ~L1417–1455)

- [ ] **Step 7.1: Replace the inline blue info-banner JSX with the warning component.**

Find the existing block at ~L1417–1455 (rendering `envLimitation` with three branches). The replacement maps the existing tri-state cleanly:

| `envLimitation` | New variant |
|---|---|
| `{ type: "unrestricted" }` | `binding-all` |
| `{ type: "restricted", environments: [...] }` (length > 0) | `binding-some` (env badges rendered separately) |
| `{ type: "restricted", environments: [] }` | `binding-none` |

Derive the kind once for the row:

```ts
const bindingKind = getRoleEnvironmentLimitationKind(binding.role);
```

(`bindingKind` is guaranteed defined here because `envLimitation` is truthy ⇔ kind is defined — the new `membersPageEnvironment` predicate already enforces this. Add a one-line comment if the assertion isn't obvious.)

Replace the banner JSX:

Single render switch — flat tri-state, no compound `&&` chains. Render `EnvironmentLabel` directly (no `Badge` wrapper — `EnvironmentLabel` is already a styled chip; the previous `Badge` wrap was a double-pill carried over from the old blue banner):

```tsx
{envLimitation && bindingKind && (
  <div className="mx-4 mt-3">
    {(() => {
      if (envLimitation.type === "unrestricted") {
        return <DDLWarningCallout type="binding-all" kind={bindingKind} />;
      }
      if (envLimitation.environments.length === 0) {
        return <DDLWarningCallout type="binding-none" kind={bindingKind} />;
      }
      return (
        <div className="flex flex-col gap-y-2">
          <DDLWarningCallout type="binding-some" kind={bindingKind} />
          <div className="flex flex-wrap gap-1">
            {envLimitation.environments.map((env) => (
              <EnvironmentLabel
                key={env}
                environmentName={env}
                className="text-xs"
              />
            ))}
          </div>
        </div>
      );
    })()}
  </div>
)}
```

Two design choices baked in:
- The IIFE replaces three sequential compound-`&&` branches with one mutually-exclusive return chain. Reads as a flat tri-state at a glance.
- `EnvironmentLabel` renders directly without `<Badge variant="secondary">`. The previous wrap was a double-pill (Badge styles + EnvironmentLabel's own colored chip). Task 8 already renders it this way — Task 7 now matches.

Remove now-unused imports (`Info` icon, `Badge` if it's no longer used elsewhere in the file — `pnpm --dir frontend fix` will surface unused imports).

- [ ] **Step 7.2: Add an automated test for the banner tri-state.**

Pick the closest existing `MembersPage` test (or, if none exists, scaffold one in `frontend/src/react/pages/settings/MembersPage.test.tsx` using `IssueDetailRoleGrantDetails.test.tsx` as the harness reference). Add three cases that mount the member-list with one binding each and assert the rendered copy:

```ts
test("binding-all: renders DDL/DML in ALL environments warning, no env badges", () => {
  // Mount with envLimitation = { type: "unrestricted" } and role = sql-editor.
  // Assert: text "project.members.ddl-current-all" is present and the env-badge
  // container is absent.
});

test("binding-some: renders 'in the listed environments' warning + EnvironmentLabel chips", () => {
  // Mount with envLimitation = { type: "restricted", environments: ["environments/prod"] }.
  // Assert: text "project.members.ddl-current-some" present; EnvironmentLabel
  // chip for "environments/prod" rendered.
});

test("binding-none: renders 'not allowed in any environment' warning, no env badges", () => {
  // Mount with envLimitation = { type: "restricted", environments: [] }.
  // Assert: text "project.members.ddl-current-none" present.
});
```

These lock the IIFE's three return branches against future regression. Match the mocking style established in `RequestRoleSheet.test.tsx`.

- [ ] **Step 7.3: Type-check + lint.**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`
Expected: passes. The file no longer references `project.members.allow-ddl` / `*-all-environments` / `disallow-*` keys.

- [ ] **Step 7.4: Commit.**

```bash
git add frontend/src/react/pages/settings/MembersPage.tsx \
        frontend/src/react/pages/settings/MembersPage.test.tsx
git commit -m "feat(role-grant): restyle member-list binding banner as warning

Replaces the blue info banner with a permission-aware yellow warning.
Same tri-state (all envs / specific envs / no envs); copy now centers
on the risk: DDL/DML statements run directly in SQL Editor without
approval.

Refs: BYT-9390"
```

---

## Chunk 3: Issue page + locale sweep + final verification

### Task 8: Add Environments row + warning banner to `IssueDetailRoleGrantDetails`

**Files:**
- Modify: `frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx`
- Create: `frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.test.tsx`

- [ ] **Step 8.1: Write the failing test.**

```tsx
// IssueDetailRoleGrantDetails.test.tsx
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueDetailRoleGrantDetails } from "./IssueDetailRoleGrantDetails";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) => {
      let s = key;
      if (vars)
        for (const [k, v] of Object.entries(vars))
          s = s.replace(`{{${k}}}`, String(v));
      return s;
    },
  }),
}));

vi.mock("@/components/ProjectMember/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { name: "environments/prod", title: "Prod" },
    { name: "environments/test", title: "Test" },
  ],
}));

// Stub EnvironmentLabel — the real component pulls in useEnvironment +
// usePlanFeature + theme tokens we don't need to exercise here. Stub
// with a span that surfaces the env name so the test can assert on it.
vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({
    environmentName,
    className,
  }: {
    environmentName: string;
    className?: string;
  }) => (
    <span data-testid="env-label" className={className}>
      {environmentName}
    </span>
  ),
}));

// vi.mock factories are hoisted above imports — wrap the mutable test
// context in vi.hoisted() so the closure reads from a binding that's
// guaranteed initialized at hoist time. Pattern matches other tests in
// this codebase that share state between vi.mock and per-case setup.
const { mockContextRef } = vi.hoisted(() => ({
  mockContextRef: { current: undefined as unknown },
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => mockContextRef.current,
}));

beforeEach(() => {
  // Force every test to set its own context — catches cases where a test
  // relies on a previous case's mock leaking through.
  mockContextRef.current = undefined;
});

// stub other stores + utility deps with the minimum shape the component
// needs (databaseStore.getDatabaseByName, roleStore.getRoleByName,
// convertFromCELString, etc.) — copy the existing pattern from
// IssueDetailDatabaseChangeView.test.tsx and trim to what the component uses.

describe("IssueDetailRoleGrantDetails", () => {
  test("renders Environments row and warning when role has DDL/DML and env list is non-empty", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: {
            expression:
              'resource.environment_id in ["prod", "test"]',
          },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      await screen.findByText(/issue.role-grant.ddl-warning/),
    ).toBeInTheDocument();
    // Mock returns translation keys verbatim — assert against the key,
    // not the English string ("Environments").
    expect(screen.getByText(/common.environments/)).toBeInTheDocument();
    // Assert env interpolation flowed into the warning text. The mock t()
    // appends a JSON-encoded vars suffix, so the warning's text content
    // contains the env titles.
    const warning = await screen.findByText(/issue.role-grant.ddl-warning/);
    expect(warning.textContent).toContain("Prod");
    expect(warning.textContent).toContain("Test");
  });

  test("hides warning + env row when role has DDL/DML but condition has no environments", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: { expression: "" },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      screen.queryByText(/issue.role-grant.ddl-warning/),
    ).not.toBeInTheDocument();
  });

  test("hides warning when role has been deleted (helper returns undefined)", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/wasDeleted",
          condition: {
            expression: 'resource.environment_id in ["prod"]',
          },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    // Helper mock returns undefined for unknown roles — warning hidden,
    // rest of the details card still renders.
    expect(
      screen.queryByText(/issue.role-grant.ddl-warning/),
    ).not.toBeInTheDocument();
  });

  test("hides warning when role lacks DDL/DML perms", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/queryOnly",
          condition: {
            expression: 'resource.environment_id in ["prod"]',
          },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      screen.queryByText(/issue.role-grant.ddl-warning/),
    ).not.toBeInTheDocument();
  });
});
```

The test stubs the role-store, env-store, and CEL-parsing helpers. Use the existing `IssueDetailDatabaseChangeView.test.tsx` as the harness reference for the stub shapes — copy the patterns rather than re-deriving them.

- [ ] **Step 8.2: Run the test and confirm it fails.**

Run: `pnpm --dir frontend test -- IssueDetailRoleGrantDetails.test.tsx`
Expected: FAIL — warning + env row not rendered.

- [ ] **Step 8.3: Update `IssueDetailRoleGrantDetails.tsx`.**

Edits inside the component (between the existing role + permissions block and the existing database + expiration blocks):

1. Add imports (and extend the existing `react` import to include `useMemo`):

```ts
// Extend the existing import — file currently imports useEffect, useState.
import { useEffect, useMemo, useState } from "react";

import { getRoleEnvironmentLimitationKind } from "@/components/ProjectMember/utils";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
import { useEnvironmentList } from "@/react/hooks/useAppState";
```

2. After the existing `condition` `useEffect` block, derive:

```ts
const envKind = getRoleEnvironmentLimitationKind(requestRoleName);
const envNames = condition?.environments ?? [];
const envList = useEnvironmentList();
const envTitles = useMemo(() => {
  const names = condition?.environments ?? [];
  const byName = new Map(envList.map((e) => [e.name, e.title]));
  return names.map((n) => byName.get(n) ?? n);
}, [condition?.environments, envList]);
```

`useEnvironmentList` is the React-native hook for the environment store (already used by `EnvironmentLabel.tsx`). The memo depends on the **stable upstream** `condition?.environments` (set by the async `useEffect` parsing the CEL string), not on the locally-derived `envNames` — `envNames = condition?.environments ?? []` produces a fresh `[]` on every render whenever `condition` is undefined, which would force the memo to re-run unconditionally. Falls back to the raw env resource name (e.g. `environments/prod-old`) if the env isn't in the store, which can happen if the env was renamed or deleted between request submission and approver review — preserves visibility instead of dropping the row silently.

3. As the **first child** of the bordered details `<div>` (before the role row), insert:

```tsx
{envKind && envNames.length > 0 && (
  <DDLWarningCallout
    type="issue"
    kind={envKind}
    environments={envTitles}
  />
)}
```

The `envKind && envNames.length > 0` guard narrows `envKind` to `EnvLimitationKind` inside the branch — no non-null assertion needed.

4. Add a new Environments row between Permissions and Database, rendered only when `envNames.length > 0`:

```tsx
{envNames.length > 0 && (
  <div className="flex flex-col gap-y-2">
    <span className="text-sm text-control-light">
      {t("common.environments")}
    </span>
    <div className="flex flex-wrap gap-1">
      {envNames.map((env) => (
        <EnvironmentLabel
          key={env}
          environmentName={env}
          className="text-xs"
        />
      ))}
    </div>
  </div>
)}
```

- [ ] **Step 8.4: Run the test and confirm it passes.**

Run: `pnpm --dir frontend test -- IssueDetailRoleGrantDetails.test.tsx`
Expected: PASS — 3 cases.

- [ ] **Step 8.5: Commit.**

```bash
git add frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.tsx \
        frontend/src/react/pages/project/issue-detail/components/IssueDetailRoleGrantDetails.test.tsx
git commit -m "feat(role-grant): show env row + warning on approver issue page

Approvers reviewing a role-grant request now see (a) the list of
environments the request scopes to and (b) a yellow warning banner at
the top of the details card naming the env titles and the affected
statement kinds (DDL / DML / DDL/DML).

Refs: BYT-9390"
```

---

### Task 9: Translate the new keys into non-English React locales

**Files:**
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/vi-VN.json`

For each non-EN React locale, add all five new keys (4 under `project.members.*`, 1 under `issue.role-grant.*`) using the translations below. The `{{kind}}` and `{{environments}}` placeholders are spliced raw — do not localize the kind string (it's a product permission name).

**zh-CN:**

```json
"project.members.ddl-warning": "在所选环境中，{{kind}} 语句可在 SQL 编辑器中直接执行，无需审批。",
"project.members.ddl-current-all": "{{kind}} 语句可在 SQL 编辑器中针对所有（ALL）环境直接执行，无需审批。",
"project.members.ddl-current-some": "{{kind}} 语句可在 SQL 编辑器中针对所列环境直接执行，无需审批。",
"project.members.ddl-current-none": "{{kind}} 语句不允许在任何环境中通过 SQL 编辑器直接执行。",
"issue.role-grant.ddl-warning": "若审批通过，在 {{environments}} 中，{{kind}} 语句可在 SQL 编辑器中直接执行，无需进一步审批。"
```

**ja-JP:**

```json
"project.members.ddl-warning": "選択した環境では、{{kind}} ステートメントを SQL エディターで承認なしに直接実行できます。",
"project.members.ddl-current-all": "すべての（ALL）環境で {{kind}} ステートメントを SQL エディターで承認なしに直接実行できます。",
"project.members.ddl-current-some": "リストされた環境で {{kind}} ステートメントを SQL エディターで承認なしに直接実行できます。",
"project.members.ddl-current-none": "{{kind}} ステートメントはどの環境でも SQL エディターで直接実行することは許可されていません。",
"issue.role-grant.ddl-warning": "承認された場合、{{environments}} で {{kind}} ステートメントを SQL エディターで追加の承認なしに直接実行できます。"
```

**es-ES:**

```json
"project.members.ddl-warning": "En los entornos seleccionados, las sentencias {{kind}} pueden ejecutarse directamente en el editor SQL sin aprobación.",
"project.members.ddl-current-all": "Las sentencias {{kind}} pueden ejecutarse directamente en el editor SQL en TODOS los entornos sin aprobación.",
"project.members.ddl-current-some": "Las sentencias {{kind}} pueden ejecutarse directamente en el editor SQL en los entornos listados sin aprobación.",
"project.members.ddl-current-none": "Las sentencias {{kind}} no se pueden ejecutar directamente en el editor SQL en ningún entorno.",
"issue.role-grant.ddl-warning": "Si se aprueba, en {{environments}}, las sentencias {{kind}} podrán ejecutarse directamente en el editor SQL sin aprobación adicional."
```

**vi-VN:**

```json
"project.members.ddl-warning": "Trong các môi trường đã chọn, câu lệnh {{kind}} có thể được chạy trực tiếp trong SQL Editor mà không cần phê duyệt.",
"project.members.ddl-current-all": "Câu lệnh {{kind}} có thể được chạy trực tiếp trong SQL Editor ở TẤT CẢ các môi trường mà không cần phê duyệt.",
"project.members.ddl-current-some": "Câu lệnh {{kind}} có thể được chạy trực tiếp trong SQL Editor ở các môi trường được liệt kê mà không cần phê duyệt.",
"project.members.ddl-current-none": "Câu lệnh {{kind}} không được phép chạy trực tiếp trong SQL Editor ở bất kỳ môi trường nào.",
"issue.role-grant.ddl-warning": "Nếu được phê duyệt, trong {{environments}}, câu lệnh {{kind}} có thể được chạy trực tiếp trong SQL Editor mà không cần phê duyệt thêm."
```

If you are not a native reader of one of these languages, treat the strings above as drafts and flag them in the PR description for native review — do not silently rewrite them.

- [ ] **Step 9.1: Add the five keys to each non-EN locale file.**

Place keys alphabetically under `project.members.*` and `issue.role-grant.*` to match Biome's sort order.

- [ ] **Step 9.2: Lint to confirm sort + JSON validity.**

Run: `pnpm --dir frontend fix && pnpm --dir frontend check`
Expected: passes; no schema/sort errors.

- [ ] **Step 9.3: Commit.**

```bash
git add frontend/src/react/locales/{zh-CN,ja-JP,es-ES,vi-VN}.json
git commit -m "i18n: translate ddl-warning + ddl-current-* into zh/ja/es/vi

Translations for all five new keys across the four non-English React
locales. Old keys are removed in the final cleanup commit.

Refs: BYT-9390"
```

---

### Task 10: Remove the old i18n keys + final verification sweep

**Files:**
- Modify: `frontend/src/react/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove `allow-ddl`, `allow-ddl-all-environments`, `disallow-ddl-all-environments`.
- Modify: `frontend/src/locales/{en-US,zh-CN,ja-JP,es-ES,vi-VN}.json` — remove `allow-ddl` only.

- [ ] **Step 10.1: Confirm no source code still references the old keys or the transitional shim.**

Run:

```bash
grep -rn "allow-ddl\|disallow-ddl-all-environments\|roleHasEnvironmentLimitation" \
  frontend/src --include="*.ts" --include="*.tsx" --include="*.vue"
```

Expected: zero results outside locale files. If `roleHasEnvironmentLimitation` still appears, the migration left a caller behind — fix that caller (use `getRoleEnvironmentLimitationKind`) before continuing. If old i18n keys appear in source (not in locale JSON), fix the call site first.

- [ ] **Step 10.2: Delete the keys from all 10 locale files.**

React tree (5 files): remove `project.members.allow-ddl`, `project.members.allow-ddl-all-environments`, `project.members.disallow-ddl-all-environments`.

Vue tree (5 files): remove `project.members.allow-ddl` only.

Also remove the **transitional shim** `roleHasEnvironmentLimitation` from `frontend/src/components/ProjectMember/utils.ts` (added in Task 1 to keep the build green during migration). Step 10.1's grep confirmed no callers remain.

- [ ] **Step 10.3: Lint + type-check + test sweep.**

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

All four must pass cleanly.

- [ ] **Step 10.4: Manually exercise each surface in the dev server.**

Start the backend + dev server per the project's standard instructions (`pnpm --dir frontend dev` against a local Bytebase). Verify:

1. **Admin grant drawer — DDL/DML role.** Project → Members → Grant Role for an `sqlEditorUser`-style role: yellow warning shows below the env multiselect with `DDL/DML statements`.
2. **Admin grant drawer — DDL-only role.** Use a role that has only `bb.sql.ddl`: warning copy says `DDL statements`. Confirms `variantToKey` + interpolation routing.
3. **Admin grant drawer — DML-only role.** Same with a `bb.sql.dml`-only role: copy says `DML statements`.
4. **Admin grant drawer — non-DDL role.** Switch to a query-only role: env field and warning both disappear together (no stale state from prior selection).
5. **Edit existing binding via the drawer.** Open the drawer in *edit* mode (not create) for an existing sql-editor binding: warning appears immediately on first render, not only after re-selecting the role.
6. **Developer request drawer.** Run the same DDL/DML / DDL-only / non-DDL checks: warning shows / hides / changes label identically.
7. **Approver issue page — happy path.** Submit a request-role issue with two envs selected. Open the resulting issue as an approver: yellow banner at the top of the role-grant details card lists both env titles + the correct kind label. Permissions row, new Environments row, Database row, Expired-at row all render correctly.
8. **Approver issue page — non-DDL role.** Submit a query-only role-grant request: warning + Environments row both hidden; rest of card unchanged.
9. **Approver issue page — expired binding.** View an expired role-grant issue: existing expiration UI still renders alongside the new warning; no visual collision.
10. **Member-list banner — `restricted` (envs).** For an existing binding with a sql-editor role + 2 envs, banner is yellow, copy says `… can be directly run in SQL Editor in the listed environments without approval.`, env badges render under the warning.
11. **Member-list banner — `unrestricted`.** For a binding with no env restriction: banner says `… in ALL environments without approval.` (uppercase ALL preserved), no env badges.
12. **Member-list banner — `restricted` (no envs).** For a binding restricted to an empty env list: banner says `… are not allowed to be run directly in SQL Editor in any environment.`
13. **Locale switch.** Switch UI language to each of zh-CN / ja-JP / es-ES / vi-VN: every warning above shows the translated copy with the correct `{{kind}}` and (where present) `{{environments}}` interpolation. Spot-check at least one of `restricted` and `unrestricted` per locale.

- [ ] **Step 10.5: Commit.**

```bash
git add frontend/src/react/locales/*.json frontend/src/locales/*.json \
        frontend/src/components/ProjectMember/utils.ts
git commit -m "chore(role-grant): remove obsolete keys + transitional shim

Removes the old allow-ddl* i18n keys (10 locale files) and deletes the
roleHasEnvironmentLimitation transitional shim now that all call sites
use getRoleEnvironmentLimitationKind.

Refs: BYT-9390"
```

- [ ] **Step 10.6: Push branch and open PR.**

Confirm with the user before pushing or running `gh pr create`. Per repo policy, never push or open a PR without explicit user approval.

---

## Done criteria

- All four UI surfaces show the new permission-aware yellow warning with copy framed around environment selection.
- Approver issue page shows env titles in both the dedicated row and the warning banner.
- `roleHasEnvironmentLimitation` is fully removed from the codebase; `getRoleEnvironmentLimitationKind` has unit tests.
- Old i18n keys (`allow-ddl`, `allow-ddl-all-environments`, `disallow-ddl-all-environments`) are gone from all 10 locale files; new keys are present in all 5 React locales with translations.
- `pnpm --dir frontend check && pnpm --dir frontend type-check && pnpm --dir frontend test` all pass.
- Manual smoke test of all five language × role combinations described in Step 10.4 passes.
