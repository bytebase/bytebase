<template>
  <NConfigProvider
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <ReactPageMount page="Watermark" container-class="contents" />

    <NNotificationProvider
      :max="MAX_NOTIFICATION_DISPLAY_COUNT"
      placement="bottom-right"
    >
      <OverlayStackManager>
        <NotificationContext>
          <AuthContext>
            <router-view />
          </AuthContext>
        </NotificationContext>
      </OverlayStackManager>
    </NNotificationProvider>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { Code, ConnectError } from "@connectrpc/connect";
import { cloneDeep, isEqual } from "lodash-es";
import { NConfigProvider, NNotificationProvider } from "naive-ui";
import {
  onErrorCaptured,
  onMounted,
  onUnmounted,
  watch,
  watchEffect,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import ReactPageMount from "@/react/ReactPageMount.vue";
import { dateLang, generalLang, themeOverrides } from "../naive-ui.config";
import AuthContext from "./AuthContext.vue";
import OverlayStackManager from "./components/misc/OverlayStackManager.vue";
import { overrideAppProfile } from "./customAppProfile";
import NotificationContext from "./NotificationContext.vue";
import { locale, t } from "./plugins/i18n";
import {
  type ReactQuickstartResetDetail,
  ReactShellBridgeEvent,
} from "./react/shell-bridge";
import { useNotificationStore, useUIStateStore } from "./store";
import { isDev, setDocumentTitle } from "./utils";

// Show at most 3 notifications to prevent excessive notification when shit hits the fan.
const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

const route = useRoute();
const router = useRouter();
const notificationStore = useNotificationStore();
const uiStateStore = useUIStateStore();

const handleReactLocaleChange = (event: Event) => {
  const lang = (event as CustomEvent<unknown>).detail;
  if (typeof lang === "string") {
    locale.value = lang;
    if (route.meta.title) {
      setDocumentTitle(route.meta.title(route));
    }
  }
};

const handleReactQuickstartReset = (event: Event) => {
  const keys = (event as CustomEvent<ReactQuickstartResetDetail>).detail?.keys;
  if (!Array.isArray(keys)) {
    return;
  }
  void Promise.all(
    keys
      .filter((key): key is string => typeof key === "string")
      .map((key) =>
        uiStateStore.saveIntroStateByKey({
          key,
          newState: false,
        })
      )
  );
};

onMounted(() => {
  window.addEventListener(
    ReactShellBridgeEvent.localeChange,
    handleReactLocaleChange
  );
  window.addEventListener(
    ReactShellBridgeEvent.quickstartReset,
    handleReactQuickstartReset
  );
});

onUnmounted(() => {
  window.removeEventListener(
    ReactShellBridgeEvent.localeChange,
    handleReactLocaleChange
  );
  window.removeEventListener(
    ReactShellBridgeEvent.quickstartReset,
    handleReactQuickstartReset
  );
});

watchEffect(async () => {
  // Override app profile.
  overrideAppProfile();
});

onErrorCaptured((error: unknown /* , _, info */) => {
  if (
    error instanceof ConnectError &&
    Object.values(Code).includes(error.code)
  ) {
    return;
  }

  const err = error as { response?: unknown; stack?: string };
  if (!err.response) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Internal error captured`,
      description: isDev() ? err.stack : undefined,
    });
  }
  return true;
});

// event listener for "bb.oauth.event.unknown"
// this event would be posted when an unknown state is returned by OAuth provider.
// Add it here so the notification is displayed on the main window. The OAuth callback window is short lived
// and would close before the notification has a chance to be displayed.
window.addEventListener("bb.oauth.unknown", () => {
  notificationStore.pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: t("oauth.unknown-event"),
  });
});

// Preserve specific query fields when navigating between pages.
watch(route, (current, prev) => {
  // fields is the list of query fields that we want to preserve.
  const fields = ["mode", "customTheme", "lang", "project", "filter"];
  const preservedQuery = cloneDeep(current.query);
  for (const key of fields) {
    if (preservedQuery[key] === undefined) {
      preservedQuery[key] = prev.query[key];
    }
  }
  // If the query is the same, we don't need to update the route.
  if (isEqual(current.query, preservedQuery)) {
    return;
  }
  // Otherwise, replace current route with the preserved query.
  router.replace({
    ...current,
    query: preservedQuery,
  });
});
</script>
