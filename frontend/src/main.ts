import { VueQueryPlugin } from "@tanstack/vue-query";
import "core-js/stable";
import Long from "long";
import protobufjs from "protobufjs";
import "regenerator-runtime/runtime";
import { createApp } from "vue";
import App from "./App.vue";
import "./assets/css/github-markdown-style.css";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
import "./axios";
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
import {
  humanizeTs,
  humanizeDurationV1,
  humanizeDate,
  isDev,
  isRelease,
} from "./utils";

protobufjs.util.Long = Long;
protobufjs.configure();

console.debug("dev:", isDev());
console.debug("release:", isRelease());

(async () => {
  const app = createApp(App);

  // Allow template to access various function
  app.config.globalProperties.window = window;
  app.config.globalProperties.console = console;
  app.config.globalProperties.dayjs = dayjs;
  app.config.globalProperties.humanizeTs = humanizeTs;
  app.config.globalProperties.humanizeDurationV1 = humanizeDurationV1;
  app.config.globalProperties.humanizeDate = humanizeDate;
  app.config.globalProperties.isDev = isDev();
  app.config.globalProperties.isRelease = isRelease();

  app.use(pinia);

  const initStores = async () => {
    await Promise.all([
      useActuatorV1Store().fetchServerInfo(),
      useSubscriptionV1Store().fetchSubscription(),
      useSubscriptionV1Store().fetchFeatureMatrix(),
      useAuthStore().fetchCurrentUser(),
    ]);
  };

  await initStores();

  app.use(router).use(highlight).use(i18n).use(NaiveUI).use(VueQueryPlugin);

  app.mount("#app");
})();
