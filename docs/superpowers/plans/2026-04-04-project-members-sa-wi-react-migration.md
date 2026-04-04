# Project Members, Service Accounts & Workload Identities React Migration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate project-scoped Members, Service Accounts, and Workload Identities pages from Vue to React by making the existing workspace-level React pages accept an optional `projectId` prop.

**Architecture:** The existing React pages (`MembersPage.tsx`, `ServiceAccountsPage.tsx`, `WorkloadIdentitiesPage.tsx`) gain an optional `projectId` prop. When provided, they switch to project-scoped APIs (project IAM policy, project parent resource). New Vue mount wrappers pass `projectId` from Vue Router to React. Routes swap Vue components for mount wrappers.

**Tech Stack:** React, TypeScript, Pinia stores via `useVueState`, react-i18next, Tailwind CSS

---

## File Structure

### New files
- `frontend/src/react/ProjectMembersPageMount.vue` — Vue mount wrapper
- `frontend/src/react/ProjectServiceAccountsPageMount.vue` — Vue mount wrapper
- `frontend/src/react/ProjectWorkloadIdentitiesPageMount.vue` — Vue mount wrapper

### Modified files
- `frontend/src/react/pages/settings/ServiceAccountsPage.tsx` — add `projectId` prop, project-scoped parent/permissions
- `frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx` — add `projectId` prop, project-scoped parent/permissions
- `frontend/src/react/pages/settings/MembersPage.tsx` — add `projectId` prop, project IAM policy, project roles, Request Role feature
- `frontend/src/react/pages/settings/shared/RoleMultiSelect.tsx` — add optional `scope` prop to filter role groups (project vs workspace)
- `frontend/src/router/dashboard/projectV1.ts:334-365` — swap Vue components for React mount wrappers
- `frontend/src/react/mount.ts` — no change needed (project page loaders already configured via `import.meta.glob("./pages/settings/*.tsx")`)

---

### Task 1: Service Accounts — Add project support

**Files:**
- Modify: `frontend/src/react/pages/settings/ServiceAccountsPage.tsx`

The `ServiceAccountsPage` currently uses `actuatorStore.workspaceResourceName` as parent and `hasWorkspacePermissionV2` for permissions. Add optional `projectId` prop to support project scope.

- [ ] **Step 1: Add projectId prop and project resolution**

At the top of `ServiceAccountsPage`, change the signature and add project resolution:

```tsx
export function ServiceAccountsPage({ projectId }: { projectId?: string }) {
```

Add project store and resolution after existing store declarations:

```tsx
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

// Inside the component:
const projectStore = useProjectV1Store();
const projectName = projectId ? `${projectNamePrefix}${projectId}` : undefined;
const project = useVueState(() =>
  projectName ? projectStore.getProjectByName(projectName) : undefined
);
```

- [ ] **Step 2: Switch parent resource based on scope**

Replace the existing `parent` line:

```tsx
// Before:
const parent = useVueState(() => actuatorStore.workspaceResourceName);

// After:
const parent = useVueState(() =>
  projectName ?? actuatorStore.workspaceResourceName
);
```

- [ ] **Step 3: Switch permission checks based on scope**

Replace the workspace permission check in the header button:

```tsx
// Before:
disabled={!hasWorkspacePermissionV2("bb.serviceAccounts.create")}

// After:
disabled={
  project
    ? !hasProjectPermissionV2(project, "bb.serviceAccounts.create")
    : !hasWorkspacePermissionV2("bb.serviceAccounts.create")
}
```

- [ ] **Step 4: Update session keys to include project scope**

```tsx
// Before:
sessionKey: "bb.service-accounts.active.page-size",
// After:
sessionKey: `bb.service-accounts${projectName ? `.${projectName}` : ""}.active.page-size`,
```

Same for inactive session key.

- [ ] **Step 5: Pass project to CreateServiceAccountDrawer**

Add `project` prop to the drawer:

```tsx
<CreateServiceAccountDrawer
  serviceAccount={editingSa}
  project={projectName}
  onClose={...}
  onCreated={handleCreated}
  onUpdated={handleUpdated}
/>
```

- [ ] **Step 6: Add project prop to CreateServiceAccountDrawer**

The `CreateServiceAccountDrawer` (defined inline in the same file) needs a `project` prop to scope creation and role assignment to the project. Model after `CreateWorkloadIdentityDrawer` in `frontend/src/react/components/CreateWorkloadIdentityDrawer.tsx:76-88` which already supports this.

Add `project?: string` to the drawer props. When `project` is set:
- Use `project` as parent for `createServiceAccount`
- Use `hasProjectPermissionV2` for permission checks
- Use `projectIamPolicyStore.updateProjectIamPolicy` for role assignment instead of `workspaceStore.patchIamPolicy`
- Pass `scope="project"` to `RoleMultiSelect` when in project scope (so only project + custom roles are shown)

```tsx
function CreateServiceAccountDrawer({
  serviceAccount,
  project,
  onClose,
  onCreated,
  onUpdated,
}: {
  serviceAccount: ServiceAccount | undefined;
  project?: string;
  onClose: () => void;
  onCreated: (sa: ServiceAccount) => void;
  onUpdated: (sa: ServiceAccount) => void;
}) {
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const projectEntity = useVueState(() =>
    project ? projectStore.getProjectByName(project) : undefined
  );

  const parent = useMemo(
    () => project ?? actuatorStore.workspaceResourceName,
    [project, actuatorStore]
  );

  // Permission check:
  const hasPermission = projectEntity
    ? hasProjectPermissionV2(projectEntity, isEditMode ? "bb.serviceAccounts.update" : "bb.serviceAccounts.create")
    : hasWorkspacePermissionV2(isEditMode ? "bb.serviceAccounts.update" : "bb.serviceAccounts.create");

  // In handleCreate, after creating the SA:
  if (roles.length > 0) {
    const member = getServiceAccountNameInBinding(sa.email);
    if (projectEntity) {
      // Use the same updateProjectIamPolicyForMember pattern from CreateWorkloadIdentityDrawer
      await updateProjectIamPolicyForMember(projectEntity.name, member, roles);
    } else {
      await workspaceStore.patchIamPolicy([{ member, roles }]);
    }
  }
```

Add the `updateProjectIamPolicyForMember` helper (copy from `CreateWorkloadIdentityDrawer.tsx:237-264`).

Imports to add:

```tsx
import { useProjectV1Store } from "@/store";
import { useProjectIamPolicyStore } from "@/store/modules/v1/projectIamPolicy";
import { hasProjectPermissionV2 } from "@/utils";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
```

- [ ] **Step 7: Update ServiceAccountTable permission checks for project scope**

The `ServiceAccountTable` component uses `hasWorkspacePermissionV2` for inline operations (delete, edit, undelete). Pass a `project` prop through and switch to `hasProjectPermissionV2` when in project scope.

Add to `ServiceAccountTable` props:

```tsx
function ServiceAccountTable({
  users,
  project,
  onUserUpdated,
  onUserSelected,
}: {
  users: User[];
  project?: Project;
  onUserUpdated: (user: User) => void;
  onUserSelected?: (user: User) => void;
}) {
```

Replace permission checks:

```tsx
// Before:
hasWorkspacePermissionV2("bb.serviceAccounts.delete")
// After:
(project ? hasProjectPermissionV2(project, "bb.serviceAccounts.delete") : hasWorkspacePermissionV2("bb.serviceAccounts.delete"))
```

Same for `.get` and `.undelete` permissions. Pass `project={project}` when rendering `<ServiceAccountTable>`.

- [ ] **Step 8: Run type-check and fix**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 9: Run lint and fix**

Run: `pnpm --dir frontend fix`

- [ ] **Step 10: Commit**

```bash
git add frontend/src/react/pages/settings/ServiceAccountsPage.tsx
git commit -m "feat(frontend): add project scope support to ServiceAccountsPage"
```

---

### Task 2: Workload Identities — Add project support

**Files:**
- Modify: `frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx`

Same pattern as Task 1 but simpler — the `CreateWorkloadIdentityDrawer` already supports a `project` prop.

- [ ] **Step 1: Add projectId prop and project resolution**

```tsx
export function WorkloadIdentitiesPage({ projectId }: { projectId?: string }) {
```

Add project store imports and resolution (same pattern as Task 1).

- [ ] **Step 2: Switch parent resource based on scope**

```tsx
const parent = useVueState(() =>
  projectName ?? actuatorStore.workspaceResourceName
);
```

Replace `workspaceResourceName` usage in `fetchActive` and `fetchInactive` with `parent`.

- [ ] **Step 3: Switch permission checks based on scope**

Update the create button:

```tsx
disabled={
  project
    ? !hasProjectPermissionV2(project, "bb.workloadIdentities.create")
    : !hasWorkspacePermissionV2("bb.workloadIdentities.create")
}
```

Replace `ComponentPermissionGuard` with a project-aware check. The existing `ComponentPermissionGuard` only checks workspace permissions and will block project-only users. Replace it with a conditional:

```tsx
// Before:
<ComponentPermissionGuard permissions={["bb.workloadIdentities.list"]}>
  {/* table content */}
</ComponentPermissionGuard>

// After: remove the guard wrapper. The route already has requiredPermissionList for bb.workloadIdentities.list,
// so users who reach this page already have permission. Just render the content directly.
```

- [ ] **Step 4: Update session keys to include project scope**

```tsx
sessionKey: `bb.paged-workload-identity-table${projectName ? `.${projectName}` : ""}.active`,
```

Same for inactive/deleted key.

- [ ] **Step 5: Pass project to CreateWorkloadIdentityDrawer**

Already accepts `project` prop. Just pass it:

```tsx
<CreateWorkloadIdentityDrawer
  workloadIdentity={editingWI}
  project={projectName}
  onClose={...}
  onCreated={...}
  onUpdated={...}
/>
```

- [ ] **Step 6: Update WorkloadIdentityTable permission checks**

Same pattern as Task 1 Step 7 — pass `project` prop through and switch permission checks.

- [ ] **Step 7: Run type-check and fix**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 8: Run lint and fix**

Run: `pnpm --dir frontend fix`

- [ ] **Step 9: Commit**

```bash
git add frontend/src/react/pages/settings/WorkloadIdentitiesPage.tsx
git commit -m "feat(frontend): add project scope support to WorkloadIdentitiesPage"
```

---

### Task 3: Members — Add project support

**Files:**
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx`

This is the most complex task. The workspace `MembersPage` currently uses only workspace IAM policy. Project scope needs:
- Both workspace + project policies for `getMemberBindings`
- Project IAM mutations instead of workspace
- Project role display (`projectRoleBindings`) instead of workspace roles (`workspaceLevelRoles`)
- "Request Role" button
- Description text with learn-more link

- [ ] **Step 1: Add projectId prop and project resolution**

```tsx
export function MembersPage({ projectId }: { projectId?: string }) {
```

Add imports and project resolution:

```tsx
import {
  useProjectV1Store,
  usePermissionStore,
  useRoleStore,
  useSubscriptionV1Store,
} from "@/store";
import { useProjectIamPolicyStore } from "@/store/modules/v1/projectIamPolicy";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  PRESET_WORKSPACE_ROLES,
  PresetRoleType,
  type Permission,
} from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, isBindingPolicyExpired } from "@/utils";

// Inside component:
const projectStore = useProjectV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();
const permissionStore = usePermissionStore();
const roleStore = useRoleStore();
const subscriptionStore = useSubscriptionV1Store();

const projectName = projectId ? `${projectNamePrefix}${projectId}` : undefined;
const project = useVueState(() =>
  projectName ? projectStore.getProjectByName(projectName) : undefined
);
```

- [ ] **Step 2: Fetch project IAM policy on mount**

Add a `useEffect` to fetch the project IAM policy when in project scope:

```tsx
useEffect(() => {
  if (projectName) {
    projectIamPolicyStore.getOrFetchProjectIamPolicy(projectName);
  }
}, [projectName, projectIamPolicyStore]);

const projectIamPolicy = useVueState(() =>
  projectName ? projectIamPolicyStore.getProjectIamPolicy(projectName) : undefined
);
```

- [ ] **Step 3: Update memberBindings computation for project scope**

The key difference: project scope includes both workspace + project policies and ignores workspace roles.

```tsx
const workspaceRoles = useMemo(() => new Set(PRESET_WORKSPACE_ROLES), []);

const memberBindings = useVueState(() =>
  getMemberBindings({
    policies: projectName
      ? [
          { level: "WORKSPACE" as const, policy: workspaceStore.workspaceIamPolicy },
          { level: "PROJECT" as const, policy: projectIamPolicy! },
        ]
      : [
          { level: "WORKSPACE" as const, policy: workspaceStore.workspaceIamPolicy },
        ],
    searchText: memberSearchText,
    ignoreRoles: projectName ? workspaceRoles : new Set([]),
  })
);
```

- [ ] **Step 4: Update permission check for project scope**

```tsx
const canSetIamPolicy = project
  ? hasProjectPermissionV2(project, "bb.projects.setIamPolicy")
  : hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");
```

- [ ] **Step 5: Update MemberTable to show project roles when in project scope**

Currently `MemberTable` renders `mb.workspaceLevelRoles`. In project scope, render `mb.projectRoleBindings` instead.

Add a `scope` prop to `MemberTable`:

```tsx
function MemberTable({
  bindings,
  allowEdit,
  scope,
  selectedBindings,
  onSelectionChange,
  onUpdateBinding,
  onRevokeBinding,
}: {
  bindings: MemberBinding[];
  allowEdit: boolean;
  scope: "workspace" | "project";
  selectedBindings: string[];
  onSelectionChange: (selected: string[]) => void;
  onUpdateBinding: (binding: MemberBinding) => void;
  onRevokeBinding: (binding: MemberBinding) => void;
}) {
```

In the roles column, switch based on scope:

```tsx
<td className="px-4 py-2">
  <div className="flex flex-wrap gap-1">
    {scope === "project"
      ? sortRoles(mb.projectRoleBindings.map((b) => b.role)).map((role) => (
          <Badge key={role} className="text-xs">
            {displayRoleTitle(role)}
          </Badge>
        ))
      : sortRoles([...mb.workspaceLevelRoles]).map((role) => (
          <Badge key={role} className="text-xs gap-x-1">
            <Building2 className="h-3 w-3" />
            {displayRoleTitle(role)}
          </Badge>
        ))}
  </div>
</td>
```

Add `selectDisabled` logic for project scope — members with no project role bindings can't be selected:

```tsx
// In checkbox column, disable if no project role bindings:
<input
  type="checkbox"
  checked={selectedBindings.includes(mb.binding)}
  onChange={() => toggleOne(mb.binding)}
  disabled={scope === "project" && mb.projectRoleBindings.length === 0}
/>
```

- [ ] **Step 6: Update MemberTableByRole for project scope**

Add `scope` prop to `MemberTableByRole`. In project scope, group by `projectRoleBindings` instead of `workspaceLevelRoles`:

```tsx
const roleToBindings = useMemo(() => {
  const map = new Map<string, MemberBinding[]>();
  for (const mb of bindings) {
    const roles = scope === "project"
      ? mb.projectRoleBindings.map((b) => b.role)
      : [...mb.workspaceLevelRoles];
    for (const role of roles) {
      if (!map.has(role)) map.set(role, []);
      map.get(role)!.push(mb);
    }
  }
  const sortedRoles = sortRoles([...map.keys()]);
  return sortedRoles.map((role) => ({
    role,
    members: map.get(role) ?? [],
  }));
}, [bindings, scope]);
```

Remove the `<Building2>` icon from role headers in project scope (it signifies workspace).

- [ ] **Step 7: Update mutation handlers for project scope**

In `MembersPage`, the revoke/grant operations need to target project IAM policy when in project scope.

For `handleRevokeSelected`:

```tsx
const handleRevokeSelected = async () => {
  // ... existing self-revoke check ...
  if (window.confirm(t("settings.members.revoke-access-alert"))) {
    try {
      if (projectName && projectIamPolicy) {
        const policy = structuredClone(projectIamPolicy);
        for (const binding of policy.bindings) {
          binding.members = binding.members.filter(
            (member) => !selectedMembers.includes(member)
          );
        }
        await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
      } else {
        await workspaceStore.patchIamPolicy(
          selectedMembers.map((m) => ({ member: m, roles: [] }))
        );
      }
      pushNotification({ module: "bytebase", style: "INFO", title: t("settings.members.revoked") });
      setSelectedMembers([]);
    } catch { /* error shown by store */ }
  }
};
```

For `handleMemberRevokeBinding`:

```tsx
const handleMemberRevokeBinding = async (binding: MemberBinding) => {
  try {
    if (projectName && projectIamPolicy) {
      const policy = structuredClone(projectIamPolicy);
      for (const b of policy.bindings) {
        b.members = b.members.filter((m) => m !== binding.binding);
      }
      await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
    } else {
      await workspaceStore.patchIamPolicy([{ member: binding.binding, roles: [] }]);
    }
    pushNotification({ module: "bytebase", style: "INFO", title: t("settings.members.revoked") });
  } catch { /* error shown by store */ }
};
```

- [ ] **Step 8: Update EditMemberRoleDrawer for project scope**

Pass `projectName` and `project` to the drawer. Add project IAM policy mutation support:

```tsx
function EditMemberRoleDrawer({
  member,
  projectName,
  project,
  onClose,
}: {
  member?: MemberBinding;
  projectName?: string;
  project?: Project;
  onClose: () => void;
}) {
```

Add store to component body (NOT inside handlers — hooks must be at component top level):

```tsx
const projectIamPolicyStore = useProjectIamPolicyStore();
```

Initialize `selectedRoles` from the correct scope:

```tsx
const [selectedRoles, setSelectedRoles] = useState<string[]>(() => {
  if (!member) return [];
  if (projectName) {
    // In project scope, initialize from project role bindings
    return member.projectRoleBindings.map((b) => b.role);
  }
  return [...member.workspaceLevelRoles];
});
```

In `handleSubmit`, switch between workspace and project mutations:

```tsx
if (projectName) {
  const policy = structuredClone(projectIamPolicyStore.getProjectIamPolicy(projectName));

  if (isEditMode) {
    // Remove member from all bindings, then add to selected roles
    for (const binding of policy.bindings) {
      binding.members = binding.members.filter((m) => m !== member.binding);
    }
    for (const role of selectedRoles) {
      const existing = policy.bindings.find((b) => b.role === role);
      if (existing) {
        if (!existing.members.includes(member.binding)) {
          existing.members.push(member.binding);
        }
      } else {
        policy.bindings.push(create(BindingSchema, { role, members: [member.binding] }));
      }
    }
  } else {
    for (const binding of selectedBindings) {
      for (const role of selectedRoles) {
        const existing = policy.bindings.find((b) => b.role === role);
        if (existing) {
          if (!existing.members.includes(binding)) {
            existing.members.push(binding);
          }
        } else {
          policy.bindings.push(create(BindingSchema, { role, members: [binding] }));
        }
      }
    }
  }
  await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
} else {
  // Existing workspace logic
  if (isEditMode) {
    await workspaceStore.patchIamPolicy([{ member: member.binding, roles: selectedRoles }]);
  } else {
    const batchPatch = selectedBindings.map((binding) => {
      const existedRoles = workspaceStore.findRolesByMember(binding);
      return { member: binding, roles: [...new Set([...selectedRoles, ...existedRoles])] };
    });
    await workspaceStore.patchIamPolicy(batchPatch);
  }
}
```

For `handleRevoke` in project scope:

```tsx
if (projectName) {
  const policy = structuredClone(projectIamPolicyStore.getProjectIamPolicy(projectName));
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter((m) => m !== member.binding);
  }
  await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
} else {
  await workspaceStore.patchIamPolicy([{ member: member.binding, roles: [] }]);
}
```

Import `BindingSchema` from `@/types/proto-es/v1/iam_policy_pb` and `create` from `@bufbuild/protobuf`.

Pass `scope` to `RoleMultiSelect` in the drawer so it only shows relevant roles:

```tsx
<RoleMultiSelect
  value={selectedRoles}
  onChange={setSelectedRoles}
  scope={projectName ? "project" : "workspace"}
/>
```

- [ ] **Step 9: Add scope prop to RoleMultiSelect**

Modify `frontend/src/react/pages/settings/shared/RoleMultiSelect.tsx` to accept an optional `scope` prop:

```tsx
export function RoleMultiSelect({
  value,
  onChange,
  disabled,
  scope,
}: {
  value: string[];
  onChange: (roles: string[]) => void;
  disabled?: boolean;
  scope?: "workspace" | "project";
}) {
```

In the `groups` computation, filter by scope:

```tsx
const groups = useMemo(() => {
  const kw = search.toLowerCase();
  const matchRole = (name: string) =>
    !kw || displayRoleTitle(name).toLowerCase().includes(kw);

  const workspace = scope !== "project" ? PRESET_WORKSPACE_ROLES.filter(matchRole) : [];
  const project = scope !== "workspace" ? PRESET_PROJECT_ROLES.filter(matchRole) : [];
  const custom = roleList
    .filter((r) => !PRESET_ROLES.includes(r.name))
    .map((r) => r.name)
    .filter(matchRole);
  // ... rest unchanged
```

When `scope` is omitted, all groups are shown (backward-compatible).

- [ ] **Step 10: Add project description and learn-more link**

At the top of the `MembersPage` return, before the search bar, add a project-scope description:

```tsx
{projectName && (
  <div className="textinfolabel px-4 pt-4">
    {t("project.members.description")}{" "}
    <a
      href="https://docs.bytebase.com/administration/roles/?source=console#project-roles"
      target="_blank"
      rel="noopener noreferrer"
      className="text-accent hover:underline"
    >
      {t("common.learn-more")}
    </a>
  </div>
)}
```

- [ ] **Step 10: Add Request Role button for project scope**

The Vue version shows a "Request Role" button when:
1. `project.allowRequestRole` is true
2. User has missing project owner permissions

For now, implement the visibility logic and button. The `RoleGrantPanel` is a Vue component — use it via a simple state flag that triggers a Vue-rendered panel (or implement inline if simple enough).

Add state and computed values:

```tsx
const hasRequestRoleFeature = useVueState(() =>
  subscriptionStore.hasFeature(PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW)
);

const projectOwnerPermissions = useVueState(() =>
  (roleStore.getRoleByName(PresetRoleType.PROJECT_OWNER)?.permissions ?? []) as Permission[]
);

// Must use useVueState since permissionStore is Vue-reactive
const hasMissingPermission = useVueState(() => {
  if (!project) return false;
  const currentPerms = permissionStore.currentPermissionsInProjectV1(project);
  const workspacePerms = permissionStore.currentPermissions;
  return projectOwnerPermissions.some(
    (p: Permission) => !workspacePerms.has(p) && !currentPerms.has(p)
  );
});

const shouldShowRequestRole = project?.allowRequestRole && hasMissingPermission;
```

Note: The actual "Request Role" panel creates an issue — this is a complex Vue component (`RoleGrantPanel`). For the initial migration, add the button but defer the panel implementation. The button can call `router.push` to the issue creation page with pre-filled params, or we can leave a TODO to integrate the `RoleGrantPanel` later. Check with the codebase if there's a simpler path.

- [ ] **Step 11: Pass scope props to sub-components**

Update all `MemberTable` and `MemberTableByRole` renders to pass the scope:

```tsx
<MemberTable
  bindings={memberBindings}
  allowEdit={canSetIamPolicy}
  scope={projectName ? "project" : "workspace"}
  selectedBindings={selectedMembers}
  onSelectionChange={setSelectedMembers}
  onUpdateBinding={handleMemberUpdateBinding}
  onRevokeBinding={handleMemberRevokeBinding}
/>
```

```tsx
<MemberTableByRole
  bindings={memberBindings}
  allowEdit={canSetIamPolicy}
  scope={projectName ? "project" : "workspace"}
  onUpdateBinding={handleMemberUpdateBinding}
  onRevokeBinding={handleMemberRevokeBinding}
/>
```

Pass project context to drawer:

```tsx
<EditMemberRoleDrawer
  member={editingMember}
  projectName={projectName}
  project={project}
  onClose={() => {
    setShowEditMemberDrawer(false);
    setEditingMember(undefined);
  }}
/>
```

- [ ] **Step 12: Run type-check and fix**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 13: Run lint and fix**

Run: `pnpm --dir frontend fix`

- [ ] **Step 14: Commit**

```bash
git add frontend/src/react/pages/settings/MembersPage.tsx
git commit -m "feat(frontend): add project scope support to MembersPage"
```

---

### Task 4: Create Vue mount wrappers and update routes

**Files:**
- Create: `frontend/src/react/ProjectMembersPageMount.vue`
- Create: `frontend/src/react/ProjectServiceAccountsPageMount.vue`
- Create: `frontend/src/react/ProjectWorkloadIdentitiesPageMount.vue`
- Modify: `frontend/src/router/dashboard/projectV1.ts:334-365`

- [ ] **Step 1: Create ProjectMembersPageMount.vue**

Follow the exact pattern from `frontend/src/react/ProjectGitOpsPageMount.vue`:

```vue
<template>
  <div ref="container" class="h-full" />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  projectId: string;
}>();

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

async function render() {
  if (!container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  const pageProps = { projectId: props.projectId };
  if (!root) {
    root = await mountReactPage(
      container.value,
      "MembersPage",
      pageProps
    );
  } else {
    await updateReactPage(root, "MembersPage", pageProps);
  }
}

onMounted(() => render());
watch(locale, () => render());
watch(
  () => props.projectId,
  () => render()
);
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
```

- [ ] **Step 2: Create ProjectServiceAccountsPageMount.vue**

Same template as Step 1, replacing `"MembersPage"` with `"ServiceAccountsPage"`.

- [ ] **Step 3: Create ProjectWorkloadIdentitiesPageMount.vue**

Same template as Step 1, replacing `"MembersPage"` with `"WorkloadIdentitiesPage"`.

- [ ] **Step 4: Update project routes**

In `frontend/src/router/dashboard/projectV1.ts`, update lines 334-365:

```typescript
// Members route (line 341):
// Before:
component: () => import("@/views/project/ProjectMemberDashboard.vue"),
// After:
component: () => import("@/react/ProjectMembersPageMount.vue"),

// Service Accounts route (lines 351-352):
// Before:
component: () =>
  import("@/components/User/Settings/ServiceAccountPanel.vue"),
// After:
component: () => import("@/react/ProjectServiceAccountsPageMount.vue"),

// Workload Identities route (lines 362-363):
// Before:
component: () =>
  import("@/components/User/Settings/WorkloadIdentityPanel.vue"),
// After:
component: () => import("@/react/ProjectWorkloadIdentitiesPageMount.vue"),
```

- [ ] **Step 5: Run type-check**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 6: Run lint and fix**

Run: `pnpm --dir frontend fix`

- [ ] **Step 7: Commit**

```bash
git add frontend/src/react/ProjectMembersPageMount.vue \
       frontend/src/react/ProjectServiceAccountsPageMount.vue \
       frontend/src/react/ProjectWorkloadIdentitiesPageMount.vue \
       frontend/src/router/dashboard/projectV1.ts
git commit -m "refactor(frontend): migrate project Members, Service Accounts, Workload Identities to React"
```

---

### Task 5: Verify and clean up

- [ ] **Step 1: Full type-check**

Run: `pnpm --dir frontend type-check`

- [ ] **Step 2: Full lint check**

Run: `pnpm --dir frontend check`

- [ ] **Step 3: Run tests**

Run: `pnpm --dir frontend test`

- [ ] **Step 4: Verify dev server loads correctly**

Run: `pnpm --dir frontend dev` and manually verify:
- `http://localhost:3000/members` — workspace members (unchanged behavior)
- `http://localhost:3000/service-accounts` — workspace service accounts (unchanged)
- `http://localhost:3000/workload-identities` — workspace workload identities (unchanged)
- `http://localhost:3000/projects/new-project/members` — project members (migrated)
- `http://localhost:3000/projects/new-project/service-accounts` — project service accounts (migrated)
- `http://localhost:3000/projects/new-project/workload-identities` — project workload identities (migrated)

- [ ] **Step 5: Final commit if any fixes needed**
