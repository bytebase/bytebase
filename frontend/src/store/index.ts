import { createStore, Store } from "vuex";

// Following states are persisted in database
import activity from "./modules/activity";
import message from "./modules/message";
import bookmark from "./modules/bookmark";
import member from "./modules/member";
import environment from "./modules/environment";
import project from "./modules/project";
import instance from "./modules/instance";
import dataSource from "./modules/dataSource";
import database from "./modules/database";
import sql from "./modules/sql";
import principal from "./modules/principal";
import plan from "./modules/plan";
import auth from "./modules/auth";
import issue from "./modules/issue";
import pipeline from "./modules/pipeline";
import stage from "./modules/stage";
import task from "./modules/task";

// Following states are persisted in local storage
import uistate from "./modules/uistate";

// Following states are only stored in memory
import router from "./modules/router";
import command from "./modules/command";
import notification from "./modules/notification";

// Debug module
import debug from "./modules/debug";

const isProd = import.meta.env.NODE_ENV == "production";

export const store: Store<any> = createStore({
  modules: {
    activity,
    message,
    bookmark,
    member,
    environment,
    project,
    instance,
    dataSource,
    database,
    sql,
    principal,
    plan,
    auth,
    issue,
    pipeline,
    stage,
    task,
    uistate,
    router,
    command,
    notification,
    debug,
  },
  strict: !isProd,
});
