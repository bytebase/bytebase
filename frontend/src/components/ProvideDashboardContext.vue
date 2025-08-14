<template>
  <slot v-if="!isInitializing" />
  <MaskSpinner v-else class="!bg-white" />
</template>

<script lang="ts" setup>
import { ref, onMounted } from "vue";
import { onUnmounted } from "vue";
import { useRouter } from "vue-router";
import {
  AUTH_MFA_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "@/router/auth";
import {
  useEnvironmentV1Store,
  usePolicyV1Store,
  useSettingV1Store,
  useUIStateStore,
} from "@/store";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import MaskSpinner from "./misc/MaskSpinner.vue";

const router = useRouter();
const isInitializing = ref<boolean>(true);

const policyStore = usePolicyV1Store();

let unregisterBeforeEachHook: (() => void) | undefined;
onMounted(async () => {
  await router.isReady();

  // Prepare roles, workspace policies and settings first.
  await Promise.all([
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    useSettingV1Store().fetchSettingList(),
  ]);

  // Then prepare the other resources.
  await Promise.all([useEnvironmentV1Store().fetchEnvironments()]);

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
