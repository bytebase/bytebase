import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import {
  authServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import { router } from "@/router";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { userNamePrefix, workspaceNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL } from "@/types";
import { SwitchWorkspaceRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import {
  BindingSchema,
  GetIamPolicyRequestSchema,
  IamPolicySchema,
  SetIamPolicyRequestSchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";
import { UpdateWorkspaceRequestSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { getUserListInBinding, isBindingPolicyExpired } from "@/utils";
import { useActuatorV1Store } from "./actuator";
import { composePolicyBindings } from "./projectIamPolicy";

// Notify other tabs when the user switches workspace.
const workspaceSwitchChannel = new BroadcastChannel("bb-workspace-switch");
workspaceSwitchChannel.onmessage = () => {
  // Another tab switched workspace — full-reload to the landing page
  // to pick up the new cookie and reset all frontend state.
  const landingPath = router.resolve({ name: WORKSPACE_ROUTE_LANDING }).href;
  window.location.href = landingPath;
};

export const useWorkspaceV1Store = defineStore("workspace_v1", () => {
  const _workspaceIamPolicy = ref<IamPolicy>(create(IamPolicySchema, {}));
  const workspaceList = ref<Workspace[]>([]);
  const currentWorkspace = ref<Workspace | undefined>();
  const actuatorStore = useActuatorV1Store();

  const workspaceIamPolicy = computed(() => {
    return _workspaceIamPolicy.value;
  });

  watch(
    () => actuatorStore.workspaceResourceName,
    async (name) => {
      const requestName = name || `${workspaceNamePrefix}-`;
      const workspace = await workspaceServiceClientConnect.getWorkspace({
        name: requestName,
      });
      currentWorkspace.value = workspace;
    },
    { immediate: true }
  );

  const updateWorkspace = async (
    workspace: Workspace,
    updateMask: string[]
  ) => {
    const updated = await workspaceServiceClientConnect.updateWorkspace(
      create(UpdateWorkspaceRequestSchema, {
        workspace: workspace,
        updateMask: create(FieldMaskSchema, {
          paths: updateMask,
        }),
      })
    );
    const index = workspaceList.value.findIndex(
      (ws) => ws.name === updated.name
    );
    if (index >= 0) {
      workspaceList.value[index] = updated;
    }
    currentWorkspace.value = updated;
  };

  // roleMapToUsers returns Map<roles/{role}, Set<userfullname>>
  // the user fullname can be
  // - users/{email}, includes users/ALL_USERS_USER_EMAIL
  // - serviceAccounts/{email}
  // - workloadIdentities/{email}
  const roleMapToUsers = computed(() => {
    const map = new Map<string, Set<string>>();
    for (const binding of _workspaceIamPolicy.value.bindings) {
      if (!map.has(binding.role)) {
        map.set(binding.role, new Set());
      }
      for (const fullname of getUserListInBinding({
        binding,
        ignoreGroup: false,
      })) {
        map.get(binding.role)?.add(fullname);
      }
    }
    return map;
  });

  // userMapToRoles returns Map<userfullname, Set<roles/{role}>>
  // the user fullname can be
  // - users/{email}, includes users/ALL_USERS_USER_EMAIL
  // - serviceAccounts/{email}
  // - workloadIdentities/{email}
  const userMapToRoles = computed(() => {
    const map = new Map<string, Set<string>>();
    for (const binding of _workspaceIamPolicy.value.bindings) {
      for (const fullname of getUserListInBinding({
        binding,
        ignoreGroup: false,
      })) {
        if (!map.has(fullname)) {
          map.set(fullname, new Set());
        }
        map.get(fullname)?.add(binding.role);
      }
    }
    return map;
  });

  const fetchIamPolicy = async () => {
    const request = create(GetIamPolicyRequestSchema, {
      resource: actuatorStore.workspaceResourceName,
    });
    const policy = await workspaceServiceClientConnect.getIamPolicy(request);
    await composePolicyBindings(policy.bindings);
    _workspaceIamPolicy.value = policy;
    return policy;
  };

  const mergeBinding = ({
    member,
    roles,
    policy,
  }: {
    member: string;
    roles: string[];
    policy: IamPolicy;
  }) => {
    const newRolesSet = new Set(roles);
    const workspacePolicy = cloneDeep(policy);

    for (const binding of workspacePolicy.bindings) {
      const index = binding.members.findIndex((m) => m === member);
      if (!newRolesSet.has(binding.role)) {
        if (index >= 0) {
          binding.members.splice(index, 1);
        }
      } else {
        if (index < 0) {
          binding.members.push(member);
        }
      }

      newRolesSet.delete(binding.role);
    }

    for (const role of newRolesSet) {
      workspacePolicy.bindings.push(
        create(BindingSchema, {
          role,
          members: [member],
        })
      );
    }

    return workspacePolicy;
  };

  const patchIamPolicy = async (
    batchPatch: {
      member: string;
      roles: string[];
    }[]
  ) => {
    if (batchPatch.length === 0) {
      return;
    }
    let workspacePolicy = cloneDeep(_workspaceIamPolicy.value);
    for (const patch of batchPatch) {
      workspacePolicy = mergeBinding({
        ...patch,
        policy: workspacePolicy,
      });
    }

    const request = create(SetIamPolicyRequestSchema, {
      resource: actuatorStore.workspaceResourceName,
      policy: workspacePolicy,
      etag: workspacePolicy.etag,
    });
    const policy = await workspaceServiceClientConnect.setIamPolicy(request);
    _workspaceIamPolicy.value = policy;
  };

  const findRolesByMember = (member: string): string[] => {
    const roles = new Set<string>();
    for (const binding of _workspaceIamPolicy.value.bindings) {
      if (isBindingPolicyExpired(binding)) {
        continue;
      }
      if (binding.members.includes(member)) {
        roles.add(binding.role);
      }
    }
    return [...roles];
  };

  const getWorkspaceRolesByName = (name: string) => {
    const specificRoles = userMapToRoles.value.get(name) ?? new Set<string>([]);
    if (userMapToRoles.value.has(`${userNamePrefix}${ALL_USERS_USER_EMAIL}`)) {
      for (const role of userMapToRoles.value.get(
        `${userNamePrefix}${ALL_USERS_USER_EMAIL}`
      )!) {
        specificRoles.add(role);
      }
    }
    return specificRoles;
  };

  const fetchCurrentWorkspace = async () => {
    const name =
      actuatorStore.workspaceResourceName || `${workspaceNamePrefix}-`;
    const workspace = await workspaceServiceClientConnect.getWorkspace({
      name,
    });
    currentWorkspace.value = workspace;
  };

  const fetchWorkspaceList = async () => {
    const resp = await workspaceServiceClientConnect.listWorkspaces({});
    workspaceList.value = resp.workspaces;
  };

  const switchWorkspace = async (workspaceName: string) => {
    await authServiceClientConnect.switchWorkspace(
      create(SwitchWorkspaceRequestSchema, {
        workspace: workspaceName,
        web: true,
      })
    );
    // Notify other tabs to reload with the new workspace.
    workspaceSwitchChannel.postMessage(workspaceName);
    // Full-reload to the landing page to reset all frontend state.
    // Reloading the current URL would fail if the page is project-scoped
    // (the project likely doesn't exist in the target workspace).
    const landingPath = router.resolve({ name: WORKSPACE_ROUTE_LANDING }).href;
    window.location.href = landingPath;
  };

  return {
    userMapToRoles,
    workspaceIamPolicy,
    fetchIamPolicy,
    patchIamPolicy,
    findRolesByMember,
    roleMapToUsers,
    getWorkspaceRolesByName,
    workspaceList,
    currentWorkspace,
    updateWorkspace,
    fetchCurrentWorkspace,
    fetchWorkspaceList,
    switchWorkspace,
  };
});
