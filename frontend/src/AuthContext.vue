<template>
  <slot></slot>
  <template v-if="!isAuthRoute && authStore.isLoggedIn">
    <!-- Do not show the modal when the user is in auth related pages. -->
    <SigninModal v-if="authStore.unauthenticatedOccurred" />
    <InactiveRemindModal v-else />
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { isAuthRelatedRoute } from "@/utils/auth";
import InactiveRemindModal from "@/views/auth/InactiveRemindModal.vue";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useAuthStore,
  useCurrentUserV1,
  useGroupStore,
  useRoleStore,
  useWorkspaceV1Store,
} from "./store";
import { isDev } from "./utils";

// This interval is used to check if the user's session is still valid.
// In development, it checks every minute, while in production, it checks every 5 minutes.
const CHECK_AUTHORIZATION_INTERVAL = isDev() ? 60 * 1000 : 60 * 1000 * 5;

const router = useRouter();
const authStore = useAuthStore();
const currentUser = useCurrentUserV1();
const workspaceStore = useWorkspaceV1Store();
const roleStore = useRoleStore();
const groupStore = useGroupStore();

const authCheckIntervalId = ref<NodeJS.Timeout>();

const isAuthRoute = computed(() => {
  return (
    router.currentRoute.value.name &&
    isAuthRelatedRoute(router.currentRoute.value.name.toString())
  );
});

onMounted(() => {
  // Periodically checks if the user's session is still valid.
  // Skips check if user is not logged in or on an auth-related route.
  authCheckIntervalId.value = setInterval(async () => {
    if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
      return;
    }
    if (isAuthRoute.value) {
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

watch(
  () => authStore.isLoggedIn,
  async () => {
    if (!authStore.isLoggedIn || !currentUser.value) {
      return;
    }

    await Promise.all([
      workspaceStore.fetchIamPolicy(),
      roleStore.fetchRoleList(),
      // we only care about the groups for the current user.
      groupStore.batchGetOrFetchGroups(currentUser.value.groups),
    ]);
  },
  {
    immediate: true,
  }
);
</script>
