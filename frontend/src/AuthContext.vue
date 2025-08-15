<template>
  <slot></slot>
  <SigninModal />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { groupBindingPrefix } from "@/types";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { isAuthRelatedRoute } from "@/utils/auth";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { AUTH_PASSWORD_RESET_MODULE } from "./router/auth";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import {
  useAuthStore,
  pushNotification,
  useWorkspaceV1Store,
  useGroupStore,
  useRoleStore,
} from "./store";
import { isDev } from "./utils";

// This interval is used to check if the user's session is still valid.
// In development, it checks every minute, while in production, it checks every 5 minutes.
const CHECK_AUTHORIZATION_INTERVAL = isDev() ? 60 * 1000 : 60 * 1000 * 5;

const router = useRouter();
const authStore = useAuthStore();
const workspaceStore = useWorkspaceV1Store();
const groupStore = useGroupStore();
const roleStore = useRoleStore();

const authCheckIntervalId = ref<NodeJS.Timeout>();

onMounted(() => {
  // Periodically checks if the user's session is still valid.
  // Skips check if user is not logged in or on an auth-related route.
  authCheckIntervalId.value = setInterval(async () => {
    if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
      return;
    }
    if (
      router.currentRoute.value.name &&
      isAuthRelatedRoute(router.currentRoute.value.name.toString())
    ) {
      return;
    }

    await authStore.fetchCurrentUser();
  }, CHECK_AUTHORIZATION_INTERVAL);
});

onUnmounted(() => {
  // Clean up the interval when component is unmounted.
  if (authCheckIntervalId.value) {
    clearInterval(authCheckIntervalId.value);
  }
});

// When current user changed, we need to redirect to the workspace root page.
watch(
  () => authStore.currentUserName,
  (currentUserName, prevCurrentUserName) => {
    if (
      currentUserName &&
      prevCurrentUserName &&
      currentUserName !== prevCurrentUserName
    ) {
      router.push({
        name: WORKSPACE_ROOT_MODULE,
        // to force route reload
        query: { _r: Date.now() },
      });
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("auth.login-as-another.title"),
        description: t("auth.login-as-another.content"),
      });
    }
  }
);

const getGroupList = (policy: IamPolicy) => {
  return policy.bindings.reduce((list, binding) => {
    for (const member of binding.members) {
      if (member.startsWith(groupBindingPrefix)) {
        list.push(member);
      }
    }
    return list;
  }, [] as string[]);
};

watch(
  () => authStore.isLoggedIn,
  async () => {
    if (!authStore.isLoggedIn) {
      return;
    }

    const [policy, _] = await Promise.all([
      workspaceStore.fetchIamPolicy(),
      roleStore.fetchRoleList(),
    ]);

    // Only load groups that are referenced in IAM policy bindings
    // Components that need ALL groups will call fetchGroupList() separately
    const groups = getGroupList(policy);
    await groupStore.batchFetchGroups(groups);

    // If the user is required to reset their password, redirect them to the reset password page.
    if (authStore.requireResetPassword) {
      router.replace({
        name: AUTH_PASSWORD_RESET_MODULE,
      });
      return;
    }
  },
  {
    immediate: true,
  }
);
</script>
