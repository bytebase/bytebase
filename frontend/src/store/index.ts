import { createStore, Store } from "vuex";
import createPersistedState from "vuex-persistedstate";

/**
 * Following states are persisted in database
 * activity | actuator | anomaly | auth | backup | bookmark |
 * command | database | dataSource | environment | gitlab |
 * inbox |instance | issue | issueSubscriber | member | notification |
 * pipeline | plan | policy | principal | project | projectWebhook | repository |
 *
 * Following states are only stored in memory
 * router | setting | sql | stage | table | task | sqlEditor
 *
 * Following states are persisted in local storage
 * uistate | vsc | view
 */
import modules from "./modules/index";

const isProd = import.meta.env.NODE_ENV == "production";

export const store: Store<any> = createStore({
  modules,
  plugins: [
    createPersistedState({
      paths: ["editorSelector"],
    }),
  ],
  strict: !isProd,
});
