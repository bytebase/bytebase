import { createApp } from "vue";
import axios from "axios";
import moment from "moment";
// @ts-ignore
import VueClipboard from "vue3-clipboard";
import App from "./App.vue";
import { store } from "./store";
import { router } from "./router";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
// @ts-ignore
import { makeServer } from "./miragejs/server";
import {
  BBAlert,
  BBAvatar,
  BBButtonAdd,
  BBButtonConfirm,
  BBCheckbox,
  BBContextMenu,
  BBModal,
  BBNotification,
  BBOutline,
  BBSelect,
  BBStepBar,
  BBSwitch,
  BBTab,
  BBTabPanel,
  BBTable,
  BBTableCell,
  BBTableHeaderCell,
  BBTableSearch,
  BBTableTabFilter,
  BBTextField,
} from "./bbkit";
import databaseSyncStatus from "./directives/database-sync-status";
import dataSourceType from "./directives/data-source-type";
// @ts-ignore
import highlight from "./directives/highlight";
import {
  isDev,
  isDemo,
  isMock,
  isRelease,
  humanizeTs,
  sizeToFit,
  urlfy,
  environmentName,
  environmentSlug,
  projectName,
  projectSlug,
  instanceName,
  instanceSlug,
  databaseSlug,
  dataSourceSlug,
  registerStoreWithRoleUtil,
} from "./utils";

if (!isRelease()) {
  makeServer();
}

const app = createApp(App);

// Allow template to access various function
app.config.globalProperties.window = window;
app.config.globalProperties.console = console;
app.config.globalProperties.moment = moment;
app.config.globalProperties.humanizeTs = humanizeTs;
app.config.globalProperties.isDev = isDev();
app.config.globalProperties.isDemo = isDemo();
app.config.globalProperties.isMock = isMock();
app.config.globalProperties.isRelease = isRelease();
app.config.globalProperties.sizeToFit = sizeToFit;
app.config.globalProperties.urlfy = urlfy;
app.config.globalProperties.environmentName = environmentName;
app.config.globalProperties.environmentSlug = environmentSlug;
app.config.globalProperties.projectName = projectName;
app.config.globalProperties.projectSlug = projectSlug;
app.config.globalProperties.instanceName = instanceName;
app.config.globalProperties.instanceSlug = instanceSlug;
app.config.globalProperties.databaseSlug = databaseSlug;
app.config.globalProperties.dataSourceSlug = dataSourceSlug;

registerStoreWithRoleUtil(store);

console.debug("dev: ", isDev());
console.debug("mode: ", import.meta.env.MODE);

axios.defaults.timeout = 3000;
axios.interceptors.request.use((request) => {
  if (request.url!.startsWith("/api")) {
    // For demo version, we always use the mock data.
    // Otherwise, we will gradually move the mock endpoint to the real backend endpoint.
    if (
      isDemo() ||
      isMock() ||
      (isDev() &&
        !request.url!.startsWith("/api/ping") &&
        !request.url!.startsWith("/api/auth/login") &&
        !request.url!.startsWith("/api/auth/signup") &&
        !request.url!.startsWith("/api/principal") &&
        !request.url!.startsWith("/api/member") &&
        !request.url!.startsWith("/api/project") &&
        !request.url!.startsWith("/api/environment") &&
        !request.url!.startsWith("/api/instance") &&
        !request.url!.startsWith("/api/database") &&
        !request.url!.startsWith("/api/issue") &&
        !request.url!.startsWith("/api/pipeline") &&
        !request.url!.startsWith("/api/activity") &&
        !request.url!.startsWith("/api/bookmar") &&
        !request.url!.startsWith("/api/sql"))
    ) {
      request.url = request.url!.replace("/api", "/mock");
    }
  }

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
  (error) => {
    if (error.response) {
      if (error.response.data.message) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: error.response.data.message,
        });
      } else {
        throw error;
      }
    }
  }
);

// TODO: A global hook to collect errors to a central service
// app.config.errorHandler = function (err, vm, info) {
// };

app
  // Need to use a directive on the element.
  // The normal hljs.initHighlightingOnLoad() won't work because router change would cause vue
  // to re-render the page and remove the event listener required for
  .directive("highlight", highlight)
  .directive("database-sync-status", databaseSyncStatus)
  .directive("data-source-type", dataSourceType)
  .use(store)
  .use(router)
  .use(VueClipboard)
  .component("BBAlert", BBAlert)
  .component("BBAvatar", BBAvatar)
  .component("BBButtonAdd", BBButtonAdd)
  .component("BBButtonConfirm", BBButtonConfirm)
  .component("BBCheckbox", BBCheckbox)
  .component("BBContextMenu", BBContextMenu)
  .component("BBModal", BBModal)
  .component("BBNotification", BBNotification)
  .component("BBOutline", BBOutline)
  .component("BBSelect", BBSelect)
  .component("BBStepBar", BBStepBar)
  .component("BBSwitch", BBSwitch)
  .component("BBTab", BBTab)
  .component("BBTabPanel", BBTabPanel)
  .component("BBTable", BBTable)
  .component("BBTableCell", BBTableCell)
  .component("BBTableHeaderCell", BBTableHeaderCell)
  .component("BBTableSearch", BBTableSearch)
  .component("BBTableTabFilter", BBTableTabFilter)
  .component("BBTextField", BBTextField)
  .mount("#app");
