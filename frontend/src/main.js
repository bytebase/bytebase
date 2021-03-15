import { createApp } from "vue";
import moment from "moment";
import highlight from "./bbkit/directives/highlight";
import App from "./App.vue";
import { store } from "./store";
import { router } from "./router";
import "./assets/css/inter.css";
import "./assets/css/tailwind.css";
import { makeServer } from "./miragejs/server";
import {
  BBAlert,
  BBAvatar,
  BBContextMenu,
  BBModal,
  BBNotification,
  BBOutline,
  BBSelect,
  BBStepBar,
  BBTab,
  BBTabPanel,
  BBTable,
  BBTableCell,
  BBTableHeaderCell,
  BBTableSearch,
  BBTableTabFilter,
} from "./bbkit";
import { isDemo, isDevOrDemo, humanizeTs, sizeToFit } from "./utils";

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

app
  // Need to use a directive on the element.
  // The normal hljs.initHighlightingOnLoad() won't work because router change would cause vue
  // to re-render the page and remove the event listener required for
  .directive("highlight", highlight)
  .use(store)
  .use(router)
  .component("BBAlert", BBAlert)
  .component("BBAvatar", BBAvatar)
  .component("BBContextMenu", BBContextMenu)
  .component("BBModal", BBModal)
  .component("BBNotification", BBNotification)
  .component("BBOutline", BBOutline)
  .component("BBSelect", BBSelect)
  .component("BBStepBar", BBStepBar)
  .component("BBTab", BBTab)
  .component("BBTabPanel", BBTabPanel)
  .component("BBTable", BBTable)
  .component("BBTableCell", BBTableCell)
  .component("BBTableHeaderCell", BBTableHeaderCell)
  .component("BBTableSearch", BBTableSearch)
  .component("BBTableTabFilter", BBTableTabFilter)
  .mount("#app");
