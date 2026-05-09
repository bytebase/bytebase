// Must be first - initializes global compatibility shims
import "./init";
import "regenerator-runtime/runtime";
import { createApp } from "vue";
import App from "./App.vue";
import "./assets/css/github-markdown-style.css";
import "./assets/css/tailwind.css";
import dayjs from "./plugins/dayjs";
import highlight from "./plugins/highlight";
import i18n from "./plugins/i18n";
import NaiveUI from "./plugins/naive-ui";
import "./polyfill";
import { router } from "./router";
import {
  pinia,
  useActuatorV1Store,
  useAuthStore,
  useSubscriptionV1Store,
  useWorkspaceV1Store,
} from "./store";
import {
  humanizeDate,
  humanizeTs,
  isDev,
  isRelease,
  migrateStorageKeys,
  migrateUIDStorageKeys,
  semverCompare,
} from "./utils";

console.debug("dev:", isDev());
console.debug("release:", isRelease());

// Migrate renamed localStorage keys before any store reads from storage.
migrateStorageKeys();

(async () => {
  const app = createApp(App);

  // Allow template to access various function
  app.config.globalProperties.window = window;
  app.config.globalProperties.console = console;
  app.config.globalProperties.dayjs = dayjs;
  app.config.globalProperties.humanizeTs = humanizeTs;
  app.config.globalProperties.humanizeDate = humanizeDate;
  app.config.globalProperties.isDev = isDev();
  app.config.globalProperties.isRelease = isRelease();

  app.use(pinia);

  const currentUser = await useAuthStore().fetchCurrentUser();
  // Initialize stores.
  const initPromises: Promise<unknown>[] = [
    useActuatorV1Store().fetchServerInfo(currentUser?.workspace),
  ];
  if (currentUser) {
    initPromises.push(useSubscriptionV1Store().fetchSubscription());
    initPromises.push(useWorkspaceV1Store().fetchWorkspaceList());
  }
  await Promise.all(initPromises);

  // Migrate old UID-scoped localStorage keys to email-scoped keys.
  // Only runs on versions < 3.15.0 (the bug window was v3.13.0–v3.14.1
  // where user.name changed from users/{uid} to users/{email} but
  // localStorage keys were not migrated).
  const serverVersion = useActuatorV1Store().version;
  if (
    currentUser?.email &&
    serverVersion &&
    serverVersion !== "development" &&
    semverCompare(serverVersion, "3.15.0", "lt")
  ) {
    migrateUIDStorageKeys(currentUser.email);
  }

  app.use(router).use(highlight).use(i18n).use(NaiveUI);

  app.mount("#app");
})();
