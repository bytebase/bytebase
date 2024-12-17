<template>
  <slot></slot>
  <SigninModal />
</template>

<script setup lang="ts">
import { useDialog, type DialogReactive } from "naive-ui";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import SigninModal from "@/views/auth/SigninModal.vue";
import { t } from "./plugins/i18n";
import { WORKSPACE_ROOT_MODULE } from "./router/dashboard/workspaceRoutes";
import { useActuatorV1Store, useAuthStore, pushNotification } from "./store";

// Check expiration every 15 sec and:
// 1. logout if expired.
// 2. refresh pages if login as another user.
const CHECK_LOGGEDIN_STATE_DURATION = 15 * 1000;

const router = useRouter();
const authStore = useAuthStore();
const actuatorStore = useActuatorV1Store();
const dialog = useDialog();
const $dialog = ref<DialogReactive | undefined>();
const prevLoggedIn = ref(false);

onMounted(() => {
  prevLoggedIn.value = authStore.isLoggedIn();

  setInterval(() => {
    if (!actuatorStore.initialized) {
      return;
    }

    const loggedIn = authStore.isLoggedIn();

    if (prevLoggedIn.value != loggedIn) {
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
    if (
      loggedIn &&
      authStore.currentUserId !== authStore.getUserIdFromCookie() &&
      $dialog.value === undefined
    ) {
      $dialog.value = dialog.create({
        title: t("auth.login-as-another.title"),
        content: t("auth.login-as-another.content"),
        type: "warning",
        autoFocus: false,
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onPositiveClick() {
          authStore.restoreUser().then(() => {
            router.push({
              name: WORKSPACE_ROOT_MODULE,
              // to force route reload
              query: { _r: Date.now() },
            });
          });
        },
        positiveText: t("common.confirm"),
        showIcon: false,
      });
    }
  }, CHECK_LOGGEDIN_STATE_DURATION);
});
</script>
