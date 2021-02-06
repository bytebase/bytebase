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
  BBStepBar,
  BBTab,
  BBTabPanel,
  BBTable,
  BBTableCell,
  BBTableTabFilter,
} from "./bbkit";

if (process.env.NODE_ENV === "development") {
  makeServer();
}

const app = createApp(App);

// Allow template to access window object
app.config.globalProperties.window = window;
// Allow template to access console object
app.config.globalProperties.console = console;
// Allow template to access moment object
app.config.globalProperties.moment = moment;

app
  .use(store)
  .use(router)
  .component("BBAlert", BBAlert)
  .component("BBAvatar", BBAvatar)
  .component("BBContextMenu", BBContextMenu)
  .component("BBModal", BBModal)
  .component("BBNotification", BBNotification)
  .component("BBStepBar", BBStepBar)
  .component("BBTab", BBTab)
  .component("BBTabPanel", BBTabPanel)
  .component("BBTable", BBTable)
  .component("BBTableCell", BBTableCell)
  .component("BBTableTabFilter", BBTableTabFilter)
  .mount("#app");
