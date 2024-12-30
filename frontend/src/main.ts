import { VueQueryPlugin } from "@tanstack/vue-query";
import axios from "axios";
import "core-js/stable";
import Long from "long";
import protobufjs from "protobufjs";
import "regenerator-runtime/runtime";
import { createApp } from "vue";
import App from "./App.vue";
import "./assets/css/github-markdown-style.css";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
import dayjs from "./plugins/dayjs";
import highlight from "./plugins/highlight";
import i18n from "./plugins/i18n";
import NaiveUI from "./plugins/naive-ui";
import { isSilent } from "./plugins/silent-request";
import "./polyfill";
import { router } from "./router";
import { pinia, pushNotification } from "./store";
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

axios.defaults.timeout = 30000;
axios.interceptors.request.use((request) => {
  if (isDev() && request.url!.startsWith("/api")) {
    console.debug(
      request.method?.toUpperCase() + " " + request.url + " request",
      JSON.stringify(request, null, 2)
    );
  }

  return request;
});

axios.interceptors.response.use(
  (response) => {
    if (isDev() && response.config.url!.startsWith("/api")) {
      console.debug(
        response.config.method?.toUpperCase() +
          " " +
          response.config.url +
          " response",
        JSON.stringify(response.data, null, 2)
      );
    }
    return response;
  },
  async (error) => {
    if (error.response) {
      if (error.response.data?.message && !isSilent()) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: error.response.data.message,
          // If server enables --debug, then the response will include the detailed error.
          description: error.response.data.error
            ? error.response.data.error
            : undefined,
        });
      }
    } else if (error.code == "ECONNABORTED" && !isSilent()) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Connecting server timeout. Make sure the server is running.",
      });
    }

    throw error;
  }
);

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

app.use(VueQueryPlugin);
app.use(pinia);

app.use(router).use(highlight).use(i18n).use(NaiveUI);
app.mount("#app");
