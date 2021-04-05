/*
 * Mirage JS guide on Routes: https://miragejs.com/docs/route-handlers/functions
 */
import configureUser from "./user";
import configureAuth from "./auth";
import configureRoleMapping from "./roleMapping";
import configureActivity from "./activity";
import configureBookmark from "./bookmark";
import configureTask from "./task";
import configureEnvironment from "./environment";
import configureInstance from "./instance";
import configureDatabase from "./database";
import configureDataSource from "./dataSource";

export const WORKSPACE_ID = 1;

export default function routes() {
  // Change this value to simulate response delay.
  // By default development environment has a 400ms delay.
  this.timing = 0;

  this.namespace = "api";

  // User
  configureUser(this);

  // Auth
  configureAuth(this);

  // RoleMapping
  configureRoleMapping(this);

  // Activity
  configureActivity(this);

  // Bookmark
  configureBookmark(this);

  // Task
  configureTask(this);

  // Environment
  configureEnvironment(this);

  // Instance
  configureInstance(this);

  // Database
  configureDatabase(this);

  // Data Source
  configureDataSource(this);
}
