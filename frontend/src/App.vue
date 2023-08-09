<template>
  <!-- it is recommended by naive-ui that we leave the local to null when the language is en -->
  <NConfigProvider
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <Watermark />

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
          <HelpDrawer />
        </KBarWrapper>
      </BBModalStack>
    </NDialogProvider>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { NConfigProvider, NDialogProvider } from "naive-ui";
import { ServerError } from "nice-grpc-common";
import { ClientError, Status } from "nice-grpc-web";
import { reactive, watchEffect, onErrorCaptured, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import HelpDrawer from "@/components/HelpDrawer";
import Watermark from "@/components/misc/Watermark.vue";
import { RouteMapList } from "@/types";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import BBModalStack from "./bbkit/BBModalStack.vue";
import { BBNotificationItem } from "./bbkit/types";
import KBarWrapper from "./components/KBar/KBarWrapper.vue";
import { t } from "./plugins/i18n";
import {
  useAuthStore,
  useNotificationStore,
  useUIStateStore,
  useHelpStore,
} from "./store";
import { isDev } from "./utils";

// Show at most 3 notifications to prevent excessive notification when shit hits the fan.
const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

// Check expiration every 30 sec and logout if expired
const CHECK_LOGGEDIN_STATE_DURATION = 30 * 1000;

const NOTIFICATION_DURATION = 6000;
const CRITICAL_NOTIFICATION_DURATION = 10000;

interface LocalState {
  notificationList: BBNotificationItem[];
  prevLoggedIn: boolean;
  helpTimer: number | null;
  RouteMapList: RouteMapList | null;
}

const authStore = useAuthStore();
const notificationStore = useNotificationStore();
const route = useRoute();
const router = useRouter();

const state = reactive<LocalState>({
  notificationList: [],
  prevLoggedIn: authStore.isLoggedIn(),
  helpTimer: null,
  RouteMapList: null,
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
  const notification = notificationStore.tryPopNotification({
    module: "bytebase",
  });
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
};

// watch route change for help
watch(
  () => route.name,
  async (routeName) => {
    const uiStateStore = useUIStateStore();
    const helpStore = useHelpStore();

    // Hide opened help drawer if route changed.
    helpStore.exitHelp();

    if (!state.RouteMapList) {
      const res = await fetch("/help/routeMapList.json");
      state.RouteMapList = await res.json();
    }

    // Clear timer after every route change.
    if (state.helpTimer) {
      clearTimeout(state.helpTimer);
      state.helpTimer = null;
    }

    const helpId = state.RouteMapList?.find(
      (pair) => pair.routeName === routeName
    )?.helpName;

    if (helpId && !uiStateStore.getIntroStateByKey(`${helpId}`)) {
      state.helpTimer = window.setTimeout(() => {
        helpStore.showHelp(helpId, true);
        uiStateStore.saveIntroStateByKey({
          key: `${helpId}`,
          newState: true,
        });
      }, 1000);
    }
  }
);

watchEffect(watchNotification);

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
