<template>
  <CustomThemeProvider>
    <Watermark />

    <NNotificationProvider
      :max="MAX_NOTIFICATION_DISPLAY_COUNT"
      placement="bottom-right"
    >
      <NDialogProvider>
        <OverlayStackManager>
          <KBarWrapper>
            <NotificationContext>
              <router-view />
            </NotificationContext>
          </KBarWrapper>
        </OverlayStackManager>
      </NDialogProvider>
    </NNotificationProvider>
  </CustomThemeProvider>
</template>

<script lang="ts" setup>
import { NDialogProvider, NNotificationProvider } from "naive-ui";
import { ServerError } from "nice-grpc-common";
import { ClientError, Status } from "nice-grpc-web";
import { reactive, onErrorCaptured } from "vue";
import { useRouter } from "vue-router";
import Watermark from "@/components/misc/Watermark.vue";
import CustomThemeProvider from "./CustomThemeProvider.vue";
import NotificationContext from "./NotificationContext.vue";
import KBarWrapper from "./components/KBar/KBarWrapper.vue";
import OverlayStackManager from "./components/misc/OverlayStackManager.vue";
import { t } from "./plugins/i18n";
import { useAuthStore, useNotificationStore } from "./store";
import { isDev } from "./utils";

// Show at most 3 notifications to prevent excessive notification when shit hits the fan.
const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

// Check expiration every 30 sec and logout if expired
const CHECK_LOGGEDIN_STATE_DURATION = 30 * 1000;

interface LocalState {
  prevLoggedIn: boolean;
}

const authStore = useAuthStore();
const notificationStore = useNotificationStore();
const router = useRouter();

const state = reactive<LocalState>({
  prevLoggedIn: authStore.isLoggedIn(),
});

setInterval(() => {
  const loggedIn = authStore.isLoggedIn();
  if (state.prevLoggedIn != loggedIn) {
    state.prevLoggedIn = loggedIn;
    if (!loggedIn) {
      authStore.logout().then(() => {
        router.push({ name: "auth.signin" });
      });
    }
  }
}, CHECK_LOGGEDIN_STATE_DURATION);

onErrorCaptured((error: any /* , _, info */) => {
  // Handle grpc request error.
  // It looks like: `{"path":"/bytebase.v1.AuthService/Login","code":2,"details":"Response closed without headers"}`
  if (
    (error instanceof ServerError || error instanceof ClientError) &&
    Object.values(Status).includes(error.code)
  ) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: error.details,
    });
  } else if (!error.response) {
    // If error has response, then we assume it's an http error and has already been
    // handled by the axios global handler.
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Internal error occurred`,
      description: isDev() ? error.stack : undefined,
    });
  }
  return true;
});

// event listener for "bb.oauth.event.unknown"
// this event would be posted when an unknown state is returned by OAuth provider.
// Add it here so the notification is displayed on the main window. The OAuth callback window is short lived
// and would close before the notification has a chance to be displayed.
window.addEventListener("bb.oauth.unknown", (event) => {
  notificationStore.pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: t("oauth.unknown-event"),
  });
});
</script>
