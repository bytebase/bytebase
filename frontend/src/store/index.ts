import { createStore, Store } from "vuex";

// Following states are persisted in database
import activity from "./modules/activity";
import bookmark from "./modules/bookmark";
import roleMapping from "./modules/roleMapping";
import environment from "./modules/environment";
import instance from "./modules/instance";
import dataSource from "./modules/dataSource";
import database from "./modules/database";
import principal from "./modules/principal";
import auth from "./modules/auth";
import task from "./modules/task";

// Following states are persisted in local storage
import uistate from "./modules/uistate";

// Following states are only stored in memory
import router from "./modules/router";
import command from "./modules/command";
import notification from "./modules/notification";

const debug = import.meta.env.NODE_ENV !== "production";

export const store: Store<any> = createStore({
  modules: {
    activity,
    bookmark,
    roleMapping,
    environment,
    instance,
    dataSource,
    database,
    principal,
    auth,
    task,
    uistate,
    router,
    command,
    notification,
  },
  strict: debug,
});
