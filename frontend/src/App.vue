<template>
  <NConfigProvider
    :key="key"
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <Watermark />

    <NNotificationProvider
      :max="MAX_NOTIFICATION_DISPLAY_COUNT"
      placement="bottom-right"
    >
      <NDialogProvider>
        <OverlayStackManager>
          <KBarWrapper>
            <NotificationContext>
              <AuthContext>
                <router-view v-if="actuatorStore.initialized" />
                <div
                  v-else
                  class="fixed inset-0 bg-white flex flex-col items-center justify-center"
                >
                  <NSpin />
                </div>
              </AuthContext>
            </NotificationContext>
          </KBarWrapper>
        </OverlayStackManager>
      </NDialogProvider>
    </NNotificationProvider>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import {
  NSpin,
  NConfigProvider,
  NDialogProvider,
  NNotificationProvider,
} from "naive-ui";
import { ServerError } from "nice-grpc-common";
import { ClientError, Status } from "nice-grpc-web";
import { onErrorCaptured, onMounted, watchEffect } from "vue";
import { watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import Watermark from "@/components/misc/Watermark.vue";
import { AUTH_PASSWORD_RESET_MODULE } from "@/router/auth";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import { provideAppRootContext } from "./AppRootContext";
import AuthContext from "./AuthContext.vue";
import NotificationContext from "./NotificationContext.vue";
import KBarWrapper from "./components/KBar/KBarWrapper.vue";
import OverlayStackManager from "./components/misc/OverlayStackManager.vue";
import { overrideAppProfile } from "./customAppProfile";
import { t } from "./plugins/i18n";
import {
  useActuatorV1Store,
  useAuthStore,
  useNotificationStore,
  useSubscriptionV1Store,
  useWorkspaceV1Store,
} from "./store";
import { isDev } from "./utils";

// Show at most 3 notifications to prevent excessive notification when shit hits the fan.
const MAX_NOTIFICATION_DISPLAY_COUNT = 3;

const route = useRoute();
const router = useRouter();
const { key } = provideAppRootContext();
const authStore = useAuthStore();
const notificationStore = useNotificationStore();
const actuatorStore = useActuatorV1Store();
const workspaceStore = useWorkspaceV1Store();

onMounted(async () => {
  const initActuator = async () => {
    actuatorStore.fetchServerInfo();
  };
  const initSubscription = async () => {
    await useSubscriptionV1Store().fetchSubscription();
  };
  const initFeatureMatrix = async () => {
    await useSubscriptionV1Store().fetchFeatureMatrix();
  };
  const restoreUser = async () => {
    await authStore.restoreUser();
  };
  const initBasicModules = async () => {
    await Promise.all([
      initActuator(),
      initFeatureMatrix(),
      initSubscription(),
      // We need to restore the basic info in order to perform route authentication.
      restoreUser(),
    ]);
  };

  await initBasicModules();
  actuatorStore.initialized = true;

  if (authStore.requireResetPassword) {
    router.replace({
      name: AUTH_PASSWORD_RESET_MODULE,
    });
  }
});

watchEffect(async () => {
  // Override app profile
  overrideAppProfile();

  if (authStore.currentUserId) {
    await workspaceStore.fetchIamPolicy();
  }
});

onErrorCaptured((error: any /* , _, info */) => {
  // Handle grpc request error.
  // It looks like: `{"path":"/bytebase.v1.AuthService/Login","code":2,"details":"Response closed without headers"}`
  if (
    (error instanceof ServerError || error instanceof ClientError) &&
    Object.values(Status).includes(error.code)
  ) {
    // Ignored: we will handle request errors in the error handler middleware of nice-grpc-web.
  } else if (!error.response) {
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
