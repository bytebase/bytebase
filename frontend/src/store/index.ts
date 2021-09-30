import { createStore, Store } from "vuex";
// Following states are persisted in database
import activity from "./modules/activity";
// Actuator module
import actuator from "./modules/actuator";
import anomaly from "./modules/anomaly";
import auth from "./modules/auth";
import backup from "./modules/backup";
import bookmark from "./modules/bookmark";
import command from "./modules/command";
import database from "./modules/database";
import dataSource from "./modules/dataSource";
import environment from "./modules/environment";
import gitlab from "./modules/gitlab";
import inbox from "./modules/inbox";
import instance from "./modules/instance";
import issue from "./modules/issue";
import issueSubscriber from "./modules/issueSubscriber";
import member from "./modules/member";
import notification from "./modules/notification";
import pipeline from "./modules/pipeline";
import plan from "./modules/plan";
import policy from "./modules/policy";
import principal from "./modules/principal";
import project from "./modules/project";
import projectWebhook from "./modules/projectWebhook";
import repository from "./modules/repository";
// Following states are only stored in memory
import router from "./modules/router";
import setting from "./modules/setting";
import sql from "./modules/sql";
import stage from "./modules/stage";
import table from "./modules/table";
import task from "./modules/task";
// Following states are persisted in local storage
import uistate from "./modules/uistate";
import vcs from "./modules/vcs";

const isProd = import.meta.env.NODE_ENV == "production";

export const store: Store<any> = createStore({
  modules: {
    activity,
    actuator,
    auth,
    bookmark,
    command,
    database,
    dataSource,
    environment,
    gitlab,
    instance,
    issue,
    issueSubscriber,
    member,
    inbox,
    notification,
    pipeline,
    plan,
    policy,
    principal,
    project,
    projectWebhook,
    repository,
    router,
    setting,
    sql,
    stage,
    table,
    backup,
    task,
    uistate,
    vcs,
    anomaly,
  },
  strict: !isProd,
});
