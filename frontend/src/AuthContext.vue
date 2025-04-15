<template>
  <slot></slot>
  <SigninModal />
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { AUTH_PASSWORD_RESET_MODULE } from "./router/auth";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import { useAuthStore, pushNotification } from "./store";

// Check expiration every 15 sec and:
// 1. logout if expired.
// 2. refresh pages if login as another user.
const CHECK_LOGGEDIN_STATE_DURATION = 15 * 1000;

const router = useRouter();
const authStore = useAuthStore();
const prevLoggedIn = ref(false);

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

  prevLoggedIn.value = authStore.isLoggedIn;

  setInterval(() => {
    const loggedIn = authStore.isLoggedIn;
    if (prevLoggedIn.value !== loggedIn) {
      prevLoggedIn.value = loggedIn;
      if (!loggedIn) {
        authStore.showLoginModal = true;
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: t("auth.token-expired-title"),
          description: t("auth.token-expired-description"),
        });
        return;
      }
    }
  }, CHECK_LOGGEDIN_STATE_DURATION);
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
</script>
