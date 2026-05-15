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

  // Mount the sibling React app hosting AgentWindow, SessionExpiredSurface,
  // and Toaster. The bb.vue-notification listener that routes pushNotification
  // into the React renderer is registered by the `./react/lib/toast`
  // side-effect import at the top of this file, so toasts queued before this
  // mount completes still render once it does.
  void (async () => {
    const { mountReactApp } = await import("./react/app/mount");
    await mountReactApp("#react-app");
  })();
})();
