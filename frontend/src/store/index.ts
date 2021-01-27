import { createStore, Store } from "vuex";
import activity from "./modules/activity";
import bookmark from "./modules/bookmark";
import project from "./modules/project";
import environment from "./modules/environment";
import repository from "./modules/repository";
import auth from "./modules/auth";
import group from "./modules/group";
import pipeline from "./modules/pipeline";
import uistate from "./modules/uistate";
import router from "./modules/router";

const debug = process.env.NODE_ENV !== "production";

export const store: Store<any> = createStore({
  modules: {
    activity,
    bookmark,
    project,
    environment,
    repository,
    auth,
    group,
    pipeline,
    uistate,
    router,
  },
  strict: debug,
});
