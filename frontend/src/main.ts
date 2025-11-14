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
} from "./store";
import { humanizeDate, humanizeTs, isDev, isRelease } from "./utils";

console.debug("dev:", isDev());
console.debug("release:", isRelease());

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

  // Initialize stores.
  await Promise.all([
    useActuatorV1Store().fetchServerInfo(),
    useSubscriptionV1Store().fetchSubscription(),
    useAuthStore().fetchCurrentUser(),
  ]);

  app.use(router).use(highlight).use(i18n).use(NaiveUI);

  app.mount("#app");
})();
