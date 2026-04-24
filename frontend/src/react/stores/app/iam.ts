import { create as createProto } from "@bufbuild/protobuf";
import {
  roleServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import { workspaceNamePrefix } from "@/react/lib/resourceName";
import { GetIamPolicyRequestSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesRequestSchema } from "@/types/proto-es/v1/role_service_pb";
import type { AppSliceCreator, IamSlice } from "./types";
import { bindingMatchesUser } from "./utils";

export const createIamSlice: AppSliceCreator<IamSlice> = (set, get) => ({
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
});
