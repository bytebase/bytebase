import { createStore, Store } from "vuex";

// Following states are persisted in database
import activity from "./modules/activity";
import bookmark from "./modules/bookmark";
import project from "./modules/project";
import environment from "./modules/environment";
import instance from "./modules/instance";
import dataSource from "./modules/dataSource";
import repository from "./modules/repository";
import auth from "./modules/auth";
import group from "./modules/group";
import pipeline from "./modules/pipeline";

// Following states are persisted in local storage
import uistate from "./modules/uistate";

// Following states are only stored in memory
import router from "./modules/router";
import notification from "./modules/notification";

const debug = process.env.NODE_ENV !== "production";

export const store: Store<any> = createStore({
  modules: {
    activity,
    bookmark,
    project,
    environment,
    instance,
    dataSource,
    repository,
    auth,
    group,
    pipeline,
    uistate,
    router,
    notification,
  },
  strict: debug,
});
