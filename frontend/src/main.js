import { createApp } from "vue";
import moment from "moment";
import VueClipboard from "vue3-clipboard";
import App from "./App.vue";
import { store } from "./store";
import { router } from "./router";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
import { makeServer } from "./miragejs/server";
import {
  BBAlert,
  BBAvatar,
  BBButtonAdd,
  BBButtonTrash,
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
} from "./bbkit";
import dataSourceType from "./directives/data-source-type";
import highlight from "./directives/highlight";
import {
  isDemo,
  isDevOrDemo,
  humanizeTs,
  sizeToFit,
  urlfy,
  instanceSlug,
  databaseSlug,
} from "./utils";

if (isDevOrDemo()) {
  makeServer();
}

const app = createApp(App);

// Allow template to access various function
app.config.globalProperties.window = window;
app.config.globalProperties.console = console;
app.config.globalProperties.moment = moment;
app.config.globalProperties.humanizeTs = humanizeTs;
app.config.globalProperties.isDemo = isDemo();
app.config.globalProperties.isDevOrDemo = isDevOrDemo();
app.config.globalProperties.sizeToFit = sizeToFit;
app.config.globalProperties.urlfy = urlfy;
app.config.globalProperties.instanceSlug = instanceSlug;
app.config.globalProperties.databaseSlug = databaseSlug;

app
  // Need to use a directive on the element.
  // The normal hljs.initHighlightingOnLoad() won't work because router change would cause vue
  // to re-render the page and remove the event listener required for
  .directive("highlight", highlight)
  .directive("data-source-type", dataSourceType)
  .use(store)
  .use(router)
  .use(VueClipboard)
  .component("BBAlert", BBAlert)
  .component("BBAvatar", BBAvatar)
  .component("BBButtonAdd", BBButtonAdd)
  .component("BBButtonTrash", BBButtonTrash)
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
  .mount("#app");
