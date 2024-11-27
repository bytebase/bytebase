import { VueQueryPlugin } from "@tanstack/vue-query";
import axios from "axios";
import isEmpty from "lodash-es/isEmpty";
import Long from "long";
import protobufjs from "protobufjs";
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
import { router } from "./router";
import { pinia, pushNotification, useAuthStore } from "./store";
import {
  humanizeTs,
  humanizeDuration,
  humanizeDurationV1,
  humanizeDate,
  isDev,
  isRelease,
  sizeToFit,
  urlfy,
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
      // When receiving 401 and is returned by our server, it means the current
      // login user's token becomes invalid. Thus we force a logout.
      // We could receive 401 when calling external service such as VCS provider,
      // in such case, we shouldn't logout.
      if (error.response.status == 401) {
        const origin = location.origin;
        const pathname = location.pathname;
        if (
          !pathname.startsWith("/auth") &&
          error.response.request.responseURL.startsWith(origin)
        ) {
          // If the request URL starts with the browser's location origin
          // e.g. http://localhost:3000/
          // we know this is a request to Bytebase API endpoint (not an external service).
          // Means that the auth session is error or expired.
          // So we need to "kick out" here.
          await useAuthStore().logout();
        }
      }

      // in such case, we shouldn't logout.
      if (error.response.status == 403) {
        const origin = location.origin;
        if (error.response.request.responseURL.startsWith(origin)) {
          // If the request URL starts with the browser's location origin
          // e.g. http://localhost:3000/
          // we know this is a request to Bytebase API endpoint (not an external service).
          // Means that the API request is denied by authorization reasons.
          router.push({ name: "error.403" });
        }
      }

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
app.config.globalProperties.humanizeDuration = humanizeDuration;
app.config.globalProperties.humanizeDurationV1 = humanizeDurationV1;
app.config.globalProperties.humanizeDate = humanizeDate;
app.config.globalProperties.isDev = isDev();
app.config.globalProperties.isRelease = isRelease();
app.config.globalProperties.sizeToFit = sizeToFit;
app.config.globalProperties.urlfy = urlfy;
app.config.globalProperties.isEmpty = isEmpty;

app.use(VueQueryPlugin);
app.use(pinia);

app.use(router).use(highlight).use(i18n).use(NaiveUI);
app.mount("#app");
