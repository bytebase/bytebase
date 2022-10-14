import axios from "axios";
import isEmpty from "lodash-es/isEmpty";
import { createApp } from "vue";

import i18n from "./plugins/i18n";
import NaiveUI from "./plugins/naive-ui";
import dayjs from "./plugins/dayjs";
import highlight from "./plugins/highlight";
import mountDemoApp from "./plugins/demo";
import { router } from "./router";
import {
  pinia,
  pushNotification,
  useActuatorStore,
  useAuthStore,
  useSubscriptionStore,
} from "./store";
import {
  databaseSlug,
  dataSourceSlug,
  environmentName,
  environmentSlug,
  humanizeTs,
  instanceName,
  instanceSlug,
  connectionSlug,
  isDev,
  isRelease,
  projectName,
  projectSlug,
  sizeToFit,
  urlfy,
} from "./utils";
import dataSourceType from "./directives/data-source-type";
import App from "./App.vue";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
import "./assets/css/github-markdown-style.css";
import "./plugins/demo/style.css";

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
        if (error.response.request.responseURL.startsWith(origin)) {
          // If the request URL starts with the browser's location origin
          // e.g. http://localhost:3000/
          // we know this is a request to Bytebase API endpoint (not an external service).
          // Means that the auth session is error or expired.
          // So we need to "kick out" here.
          try {
            await useAuthStore().logout();
          } finally {
            router.push({ name: "auth.signin" });
          }
        }
      }

      if (error.response.data.message) {
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
    } else if (error.code == "ECONNABORTED") {
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
app.config.globalProperties.isDev = isDev();
app.config.globalProperties.isRelease = isRelease();
app.config.globalProperties.sizeToFit = sizeToFit;
app.config.globalProperties.urlfy = urlfy;
app.config.globalProperties.isEmpty = isEmpty;
app.config.globalProperties.environmentName = environmentName;
app.config.globalProperties.environmentSlug = environmentSlug;
app.config.globalProperties.projectName = projectName;
app.config.globalProperties.projectSlug = projectSlug;
app.config.globalProperties.instanceName = instanceName;
app.config.globalProperties.instanceSlug = instanceSlug;
app.config.globalProperties.databaseSlug = databaseSlug;
app.config.globalProperties.dataSourceSlug = dataSourceSlug;
app.config.globalProperties.connectionSlug = connectionSlug;

app
  // Need to use a directive on the element.
  // The normal hljs.initHighlightingOnLoad() won't work because router change would cause vue
  // to re-render the page and remove the event listener required for
  .directive("data-source-type", dataSourceType)
  .use(pinia);

// We need to restore the basic info in order to perform route authentication.
// Even using the <suspense>, it's still too late, thus we do the fetch here.
// We use finally because we always want to mount the app regardless of the error.
const initActuator = () => {
  const actuatorStore = useActuatorStore();
  return actuatorStore.fetchServerInfo();
};
const initSubscription = () => {
  const subscriptionStore = useSubscriptionStore();
  return subscriptionStore.fetchSubscription();
};
const restoreUser = () => {
  const authStore = useAuthStore();
  return authStore.restoreUser();
};
Promise.all([initActuator(), initSubscription(), restoreUser()]).finally(() => {
  // Install router after the necessary data fetching is complete.
  app.use(router).use(highlight).use(i18n).use(NaiveUI);
  app.mount("#app");

  // Try to mount demo vue app instance
  const serverInfo = useActuatorStore().serverInfo;
  if ((serverInfo && serverInfo.demo && serverInfo.demoName) || isDev()) {
    mountDemoApp();
  }
});
