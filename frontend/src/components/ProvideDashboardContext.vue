<template>
  <slot v-if="!isInitializing" />
  <MaskSpinner v-else class="!bg-white" />

  <div
    v-if="!isInitializing && isSwitchingProject"
    class="fixed inset-0 z-[1000000] bg-white/50 flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { ref, onMounted } from "vue";
import { onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  AUTH_MFA_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "@/router/auth";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  useEnvironmentV1Store,
  usePolicyV1Store,
  useRoleStore,
  useSettingV1Store,
  useUserStore,
  useUIStateStore,
  useGroupStore,
  useProjectV1List,
} from "@/store";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";
import { wrapRefAsPromise } from "@/utils";
import MaskSpinner from "./misc/MaskSpinner.vue";

const route = useRoute();
const router = useRouter();
const isInitializing = ref<boolean>(true);
const isSwitchingProject = ref(false);

const policyStore = usePolicyV1Store();

const prepareProjects = async () => {
  const routeName = route.name?.toString() || "";
  if (!routeName.startsWith(`${PROJECT_V1_ROUTE_DASHBOARD}.`)) {
    await wrapRefAsPromise(useProjectV1List().ready, true);
  }
};

let unregisterBeforeEachHook: (() => void) | undefined;
onMounted(async () => {
  await router.isReady();

  // Prepare roles, workspace policies and settings first.
  await Promise.all([
    useRoleStore().fetchRoleList(),
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    useSettingV1Store().fetchSettingList(),
  ]);

  // Then prepare the other resources.
  await Promise.all([
    useUserStore().fetchUserList(),
    useGroupStore().fetchGroupList(),
    useEnvironmentV1Store().fetchEnvironments(),
    prepareProjects(),
  ]);

  useUIStateStore().restoreState();

  isInitializing.value = false;

  unregisterBeforeEachHook = router.beforeEach(async (to, from, next) => {
    if (
      to.name === AUTH_SIGNIN_MODULE ||
      to.name === AUTH_SIGNIN_ADMIN_MODULE ||
      to.name === AUTH_SIGNUP_MODULE ||
      to.name === AUTH_MFA_MODULE ||
      to.name === AUTH_PASSWORD_FORGOT_MODULE ||
      to.name === AUTH_PASSWORD_RESET_MODULE
    ) {
      next();
      return;
    }

    const fromProject = from.params.projectId as string;
    const toProject = to.params.projectId as string;
    if (fromProject !== toProject) {
      console.debug(
        `[ProvideDashboardContext] project switched ${fromProject} -> ${toProject}`
      );
      isSwitchingProject.value = true;
      if (toProject === undefined) {
        // Prepare projects if the project is not specified.
        // This is useful when the user navigates to the workspace dashboard from project detail.
        await wrapRefAsPromise(useProjectV1List().ready, true);
      }
      isSwitchingProject.value = false;
      next();
      return;
    }

    next();
  });
});

onUnmounted(() => {
  if (unregisterBeforeEachHook) {
    unregisterBeforeEachHook();
  }
});
</script>
