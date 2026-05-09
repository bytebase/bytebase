import { create as createProto } from "@bufbuild/protobuf";
import {
  projectServiceClientConnect,
  roleServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import { workspaceNamePrefix } from "@/react/lib/resourceName";
import { PRESET_WORKSPACE_ROLES } from "@/types/iam/role";
import { GetIamPolicyRequestSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesRequestSchema } from "@/types/proto-es/v1/role_service_pb";
import type { AppSliceCreator, IamSlice } from "./types";
import { bindingMatchesUser } from "./utils";

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
          set({ roles: response.roles, rolesRequest: undefined });
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
        .then((workspacePolicy) => {
          set({ workspacePolicy, workspacePolicyRequest: undefined });
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

    // Wrap the whole chain (workspace prerequisite + getIamPolicy +
    // cache write) in a single promise so the in-flight marker can be
    // set synchronously, before any await yields control to the event
    // loop. Otherwise concurrent callers that pass the cache/marker
    // check above and `await loadWorkspacePermissionState()` would each
    // issue their own `getIamPolicy` RPC after the await, since the
    // previous version only set the marker AFTER the await returned.
    const request = (async () => {
      await get().loadWorkspacePermissionState();
      try {
        const policy = await projectServiceClientConnect.getIamPolicy(
          createProto(GetIamPolicyRequestSchema, { resource: project })
        );
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
        return policy;
      } catch {
        set((state) => {
          const { [project]: _, ...projectPolicyRequests } =
            state.projectPolicyRequests;
          return { projectPolicyRequests };
        });
        return undefined;
      }
    })();
    set((state) => ({
      projectPolicyRequests: {
        ...state.projectPolicyRequests,
        [project]: request,
      },
    }));
    return request;
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
