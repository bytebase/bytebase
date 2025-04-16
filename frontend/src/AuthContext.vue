<template>
  <slot></slot>
  <SigninModal />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { isAuthRelatedRoute } from "@/utils/auth";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { AUTH_PASSWORD_RESET_MODULE } from "./router/auth";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import { useAuthStore, pushNotification, useWorkspaceV1Store } from "./store";

// Check authorization every 60 seconds.
const CHECK_AUTHORIZATION_INTERVAL = 60 * 1000;

const router = useRouter();
const authStore = useAuthStore();
const workspaceStore = useWorkspaceV1Store();

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

    const user = await authStore.fetchCurrentUser();
    if (!user) {
      authStore.unauthenticatedOccurred = true;
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("auth.token-expired-title"),
        description: t("auth.token-expired-description"),
      });
    }
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
    if (!authStore.isLoggedIn) {
      return;
    }

    await workspaceStore.fetchIamPolicy();
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
