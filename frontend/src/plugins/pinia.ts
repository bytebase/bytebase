import type { App } from "vue";
import { createPinia } from "pinia";
import localStorage from "./pinia-localstorage";

const install = (app: App) => {
  const pinia = createPinia();
  pinia.use(localStorage);
  app.use(pinia);
};

export default install;
