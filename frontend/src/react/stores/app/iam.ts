import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import {
  projectServiceClientConnect,
  roleServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import { userNamePrefix, workspaceNamePrefix } from "@/react/lib/resourceName";
import { PRESET_WORKSPACE_ROLES } from "@/types/iam/role";
import {
  BindingSchema,
  GetIamPolicyRequestSchema,
  type IamPolicy,
  IamPolicySchema,
  SetIamPolicyRequestSchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesRequestSchema } from "@/types/proto-es/v1/role_service_pb";
import { ALL_USERS_USER_EMAIL, groupBindingPrefix } from "@/types/v1/user";
import { getUserListInBinding, isBindingPolicyExpired } from "@/utils/v1/iam";
import type { AppSliceCreator, IamSlice } from "./types";
import { bindingMatchesUser } from "./utils";

// Merge a member's role set into a policy clone: drop the member from roles it
// no longer holds, add it to roles it gained, and append bindings for brand-new
// roles. Mirrors the Vue workspace store's mergeBinding.
const mergeBinding = ({
  member,
  roles,
  policy,
}: {
  member: string;
  roles: string[];
  policy: IamPolicy;
}): IamPolicy => {
  const newRolesSet = new Set(roles);
  const next = cloneDeep(policy);
  for (const binding of next.bindings) {
    const index = binding.members.findIndex((m) => m === member);
    if (!newRolesSet.has(binding.role)) {
      if (index >= 0) {
        binding.members.splice(index, 1);
      }
    } else if (index < 0) {
      binding.members.push(member);
    }
    newRolesSet.delete(binding.role);
  }
  for (const role of newRolesSet) {
    next.bindings.push(createProto(BindingSchema, { role, members: [member] }));
  }
  return next;
};

export const createIamSlice: AppSliceCreator<IamSlice> = (set, get) => ({
  projectPoliciesByName: {},
  projectPolicyRequests: {},
  roles: [],

  loadWorkspacePermissionState: async () => {
    await Promise.all([
      get().loadCurrentUser(),
      get().loadServerInfo(),
      get().loadWorkspace(),
    ]);

    const rolesRequest =
      get().rolesRequest ??
      roleServiceClientConnect
        .listRoles(createProto(ListRolesRequestSchema, {}))
        .then((response) => {
          set({
            roles: response.roles,
            roleList: response.roles,
            rolesRequest: undefined,
          });
          return response.roles;
        })
        .catch(() => {
          set({ rolesRequest: undefined });
          return [];
        });
    set({ rolesRequest });

    const policyResource =
      get().serverInfo?.workspace ||
      get().workspace?.name ||
      get().currentUser?.workspace ||
      `${workspaceNamePrefix}-`;
    const policyRequest =
      get().workspacePolicyRequest ??
      workspaceServiceClientConnect
        .getIamPolicy(
          createProto(GetIamPolicyRequestSchema, { resource: policyResource })
        )
        .then(async (workspacePolicy) => {
          set({ workspacePolicy, workspacePolicyRequest: undefined });
          // Prefetch the group / user members referenced by the policy so
          // derived role maps and any UI that resolves member display names
          // (e.g. the workspace members table's `getMemberBindings`) read
          // from a populated cache. Without the user prefetch every member
          // failed the `getUserByIdentifier` lookup and was flagged pending,
          // rendering an "Invited" badge for everyone in SaaS mode. Mirrors
          // the project IAM path in `loadProjectIamPolicy`.
          const groupMembers: string[] = [];
          const userMembers: string[] = [];
          for (const binding of workspacePolicy.bindings) {
            for (const member of binding.members) {
              if (member.startsWith(groupBindingPrefix)) {
                groupMembers.push(member);
              } else {
                userMembers.push(member);
              }
            }
          }
          await Promise.allSettled([
            get().batchGetOrFetchGroups(groupMembers),
            get().batchGetOrFetchUsers(userMembers),
          ]);
          return workspacePolicy;
        })
        .catch(() => {
          set({ workspacePolicyRequest: undefined });
          return undefined;
        });
    set({ workspacePolicyRequest: policyRequest });

    await Promise.all([rolesRequest, policyRequest]);
  },

  loadProjectIamPolicy: async (project) => {
    const existing = get().projectPoliciesByName[project];
    if (existing) return existing;
    const pending = get().projectPolicyRequests[project];
    if (pending) return pending;

    await get().loadWorkspacePermissionState();

    const request = projectServiceClientConnect
      .getIamPolicy(
        createProto(GetIamPolicyRequestSchema, { resource: project })
      )
      .then(async (policy) => {
        set((state) => {
          const { [project]: _, ...projectPolicyRequests } =
            state.projectPolicyRequests;
          return {
            projectPoliciesByName: {
              ...state.projectPoliciesByName,
              [project]: policy,
            },
            projectPolicyRequests,
          };
        });
        // Prefetch policy members into the app-store group/user caches
        // so `getMemberBindings` (in `@/react/lib/memberBindings`) can
        // resolve titles synchronously on first render. Service-account
        // and workload-identity members are fetched independently by
        // their respective settings pages — the IAM policy itself only
        // carries `user:` / `group:` prefixed members for those.
        const groupMembers: string[] = [];
        const userMembers: string[] = [];
        for (const binding of policy.bindings) {
          for (const member of binding.members) {
            if (member.startsWith(groupBindingPrefix)) {
              groupMembers.push(member);
            } else {
              userMembers.push(member);
            }
          }
        }
        await Promise.allSettled([
          get().batchGetOrFetchGroups(groupMembers),
          get().batchGetOrFetchUsers(userMembers),
        ]);
        return policy;
      })
      .catch(() => {
        set((state) => {
          const { [project]: _, ...projectPolicyRequests } =
            state.projectPolicyRequests;
          return { projectPolicyRequests };
        });
        return undefined;
      });
    set((state) => ({
      projectPolicyRequests: {
        ...state.projectPolicyRequests,
        [project]: request,
      },
    }));
    return request;
  },

  getProjectIamPolicy: (project) => {
    return (
      get().projectPoliciesByName[project] ?? createProto(IamPolicySchema, {})
    );
  },

  updateProjectIamPolicy: async (project, policy) => {
    // Dedupe members within each binding (mirrors the Pinia store's
    // pre-write normalization).
    const deduped = cloneDeep(policy);
    for (const binding of deduped.bindings) {
      if (binding.members) {
        binding.members = [...new Set(binding.members)];
      }
    }
    const updated = await projectServiceClientConnect.setIamPolicy(
      createProto(SetIamPolicyRequestSchema, {
        resource: project,
        policy: deduped,
        etag: deduped.etag,
      })
    );
    set((state) => ({
      projectPoliciesByName: {
        ...state.projectPoliciesByName,
        [project]: updated,
      },
    }));
    // Prefetch the updated policy's members into the app-store group/user
    // caches so the members table (getMemberBindings) resolves titles for
    // newly granted members without a stale render.
    const groupMembers: string[] = [];
    const userMembers: string[] = [];
    for (const binding of updated.bindings) {
      for (const member of binding.members) {
        if (member.startsWith(groupBindingPrefix)) {
          groupMembers.push(member);
        } else {
          userMembers.push(member);
        }
      }
    }
    await Promise.allSettled([
      get().batchGetOrFetchGroups(groupMembers),
      get().batchGetOrFetchUsers(userMembers),
    ]);
    return updated;
  },

  fetchWorkspaceIamPolicy: async () => {
    const resource =
      get().serverInfo?.workspace ||
      get().workspace?.name ||
      get().currentUser?.workspace ||
      `${workspaceNamePrefix}-`;
    const policy = await workspaceServiceClientConnect.getIamPolicy(
      createProto(GetIamPolicyRequestSchema, { resource })
    );
    // Prefetch groups referenced by the policy so the derived role/user maps
    // can expand group members from the same app-store cache.
    const groups = policy.bindings
      .flatMap((binding) => binding.members)
      .filter((member) => member.startsWith(groupBindingPrefix));
    if (groups.length > 0) {
      await get()
        .batchGetOrFetchGroups(groups)
        .catch(() => []);
    }
    set({ workspacePolicy: policy });
    return policy;
  },

  patchWorkspaceIamPolicy: async (batchPatch) => {
    if (batchPatch.length === 0) {
      return;
    }
    const current =
      get().workspacePolicy ?? (await get().fetchWorkspaceIamPolicy());
    let policy = cloneDeep(current);
    for (const patch of batchPatch) {
      policy = mergeBinding({ ...patch, policy });
    }
    const resource =
      get().serverInfo?.workspace ||
      get().workspace?.name ||
      get().currentUser?.workspace ||
      `${workspaceNamePrefix}-`;
    const updated = await workspaceServiceClientConnect.setIamPolicy(
      createProto(SetIamPolicyRequestSchema, {
        resource,
        policy,
        etag: policy.etag,
      })
    );
    set({ workspacePolicy: updated });
  },

  workspaceRoleMapToUsers: () => {
    const policy = get().workspacePolicy;
    const getGroupByIdentifier = get().getGroupByIdentifier;
    const map = new Map<string, Set<string>>();
    for (const binding of policy?.bindings ?? []) {
      if (!map.has(binding.role)) {
        map.set(binding.role, new Set());
      }
      for (const fullname of getUserListInBinding({
        binding,
        ignoreGroup: false,
        getGroupByIdentifier,
      })) {
        map.get(binding.role)?.add(fullname);
      }
    }
    return map;
  },

  workspaceUserMapToRoles: () => {
    const policy = get().workspacePolicy;
    const getGroupByIdentifier = get().getGroupByIdentifier;
    const map = new Map<string, Set<string>>();
    for (const binding of policy?.bindings ?? []) {
      for (const fullname of getUserListInBinding({
        binding,
        ignoreGroup: false,
        getGroupByIdentifier,
      })) {
        if (!map.has(fullname)) {
          map.set(fullname, new Set());
        }
        map.get(fullname)?.add(binding.role);
      }
    }
    return map;
  },

  findWorkspaceRolesByMember: (member) => {
    const roles = new Set<string>();
    for (const binding of get().workspacePolicy?.bindings ?? []) {
      if (isBindingPolicyExpired(binding)) {
        continue;
      }
      if (binding.members.includes(member)) {
        roles.add(binding.role);
      }
    }
    return [...roles];
  },

  getWorkspaceRolesByName: (name) => {
    const userMapToRoles = get().workspaceUserMapToRoles();
    const roles = new Set<string>(userMapToRoles.get(name) ?? []);
    const allUsersName = `${userNamePrefix}${ALL_USERS_USER_EMAIL}`;
    const allUsersRoles = userMapToRoles.get(allUsersName);
    if (allUsersRoles) {
      for (const role of allUsersRoles) {
        roles.add(role);
      }
    }
    return roles;
  },

  hasWorkspacePermission: (permission) => {
    const user = get().currentUser;
    if (!user) return false;
    const roleByName = new Map(get().roles.map((role) => [role.name, role]));
    const roleNames = bindingMatchesUser(get().workspacePolicy, user).map(
      (binding) => binding.role
    );
    return roleNames.some((roleName) =>
      roleByName.get(roleName)?.permissions.includes(permission)
    );
  },

  hasProjectPermission: (project, permission) => {
    if (get().hasWorkspacePermission(permission)) {
      return true;
    }
    const user = get().currentUser;
    if (!user) return false;
    const roleByName = new Map(get().roles.map((role) => [role.name, role]));
    const workspaceLevelProjectRoles = bindingMatchesUser(
      get().workspacePolicy,
      user
    )
      .map((binding) => binding.role)
      .filter((role) => !PRESET_WORKSPACE_ROLES.includes(role));
    const projectRoles = bindingMatchesUser(
      get().projectPoliciesByName[project.name],
      user
    ).map((binding) => binding.role);
    return [...workspaceLevelProjectRoles, ...projectRoles].some((roleName) =>
      roleByName.get(roleName)?.permissions.includes(permission)
    );
  },
});
