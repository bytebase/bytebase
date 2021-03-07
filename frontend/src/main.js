import { createApp } from "vue";
import App from "./App.vue";
import moment from "moment";
import { store } from "./store";
import { router } from "./router";
import "./index.css";
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
  BBTableTabFilter,
} from "./bbkit";
import { isDevOrDemo, humanizeTs, sizeToFit } from "./utils";

if (isDevOrDemo()) {
  makeServer();
}

const app = createApp(App);

// Allow template to access various function
app.config.globalProperties.window = window;
app.config.globalProperties.console = console;
app.config.globalProperties.moment = moment;
app.config.globalProperties.humanizeTs = humanizeTs;
app.config.globalProperties.isDevOrDemo = isDevOrDemo();
app.config.globalProperties.sizeToFit = sizeToFit;

app
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
  .component("BBTableTabFilter", BBTableTabFilter)
  .mount("#app");
