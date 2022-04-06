<template>
  <!-- it is recommended by naive-ui that we leave the local to null when the language is en -->
  <NConfigProvider
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <NDialogProvider>
      <BBModalStack>
        <KBarWrapper>
          <router-view />
          <template v-if="state.notificationList.length > 0">
            <BBNotification
              :placement="'BOTTOM_RIGHT'"
              :notification-list="state.notificationList"
              @close="removeNotification"
            />
          </template>
        </KBarWrapper>
      </BBModalStack>
    </NDialogProvider>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { reactive, watchEffect, onErrorCaptured } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";

import { isDev } from "./utils";
import { Notification } from "./types";
import { BBNotificationItem } from "./bbkit/types";
import KBarWrapper from "./components/KBar/KBarWrapper.vue";
import BBModalStack from "./bbkit/BBModalStack.vue";

import { NConfigProvider, NDialogProvider } from "naive-ui";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import { t } from "./plugins/i18n";
import { useAuthStore } from "./store";
// Show at most 3 notifications to prevent excessive notification when shit hits the fan.
const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

// Check expiration every 30 sec and logout if expired
const CHECK_LOGGEDIN_STATE_DURATION = 30 * 1000;

const NOTIFICATION_DURATION = 6000;
const CRITICAL_NOTIFICATION_DURATION = 10000;

interface LocalState {
  notificationList: BBNotificationItem[];
  prevLoggedIn: boolean;
}

const store = useStore();
const authStore = useAuthStore();
const router = useRouter();

const state = reactive<LocalState>({
  notificationList: [],
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

const removeNotification = (item: BBNotificationItem | undefined) => {
  if (!item) return;
  const index = state.notificationList.indexOf(item);
  if (index >= 0) {
    state.notificationList.splice(index, 1);
  }
};

const watchNotification = () => {
  store
    .dispatch("notification/tryPopNotification", {
      module: "bytebase",
    })
    .then((notification: Notification | undefined) => {
      if (notification) {
        if (state.notificationList.length >= MAX_NOTIFICATION_DISPLAY_COUNT) {
          state.notificationList.pop();
        }

        const item: BBNotificationItem = {
          style: notification.style,
          title: notification.title,
          description: notification.description || "",
          link: notification.link || "",
          linkTitle: notification.linkTitle || "",
        };
        state.notificationList.unshift(item);
        if (!notification.manualHide) {
          setTimeout(
            () => {
              removeNotification(item);
            },
            notification.style == "CRITICAL"
              ? CRITICAL_NOTIFICATION_DURATION
              : NOTIFICATION_DURATION
          );
        }
      }
    });
};

watchEffect(watchNotification);

onErrorCaptured((e: any /* , _, info */) => {
  // If e has response, then we assume it's an http error and has already been
  // handled by the axios global handler.
  if (!e.response) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: `Internal error occurred`,
      description: isDev() ? e.stack : undefined,
    });
  }
  return true;
});

// event listener for "bb.oauth.event.unknown"
// this event would be posted when an unknown state is returned by OAuth provider.
// Add it here so the notification is displayed on the main window. The OAuth callback window is short lived
// and would close before the notification has a chance to be displayed.
window.addEventListener("bb.oauth.unknown", (event) => {
  store.dispatch("notification/pushNotification", {
    module: "bytebase",
    style: "CRITICAL",
    title: t("oauth.unknown-event"),
  });
});
</script>
