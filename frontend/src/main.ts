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
// Side-effect: registers the bb.vue-notification window listener and
// constructs the toastManager singleton. Must load before any auth/error
// interceptor can fire pushNotification during bootstrap RPCs.
import "./react/lib/toast";
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

  app.use(router).use(highlight).use(i18n).use(NaiveUI);

  app.mount("#app");

  // Boot the React toaster after Vue is mounted. Importing mountToaster
  // pulls in @/react/lib/toast as a side effect, which registers the
  // bb.vue-notification window listener at module-eval time. Any
  // subsequent pushNotification reaches the React renderer.
  const toasterRoot = document.getElementById("bb-toaster-root");
  if (toasterRoot) {
    const { mountToaster } = await import("./react/mountToaster");
    void mountToaster(toasterRoot);
  }
})();
