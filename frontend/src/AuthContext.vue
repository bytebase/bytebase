<template>
  <slot></slot>
  <SigninModal />
</template>

<script setup lang="ts">
import { onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { AUTH_PASSWORD_RESET_MODULE } from "./router/auth";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import { useAuthStore, pushNotification, useWorkspaceV1Store } from "./store";

const router = useRouter();
const authStore = useAuthStore();
const workspaceStore = useWorkspaceV1Store();

onMounted(() => {
  if (!authStore.isLoggedIn) {
    return;
  }
  if (authStore.requireResetPassword) {
    router.replace({
      name: AUTH_PASSWORD_RESET_MODULE,
    });
    return;
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
    if (authStore.isLoggedIn) {
      await workspaceStore.fetchIamPolicy();
    }
  },
  {
    immediate: true,
  }
);
</script>
