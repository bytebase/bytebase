/*
 * Mirage JS guide on Routes: https://miragejs.com/docs/route-handlers/functions
 */
import configureUser from "./user";
import configureAuth from "./auth";
import configureRoleMapping from "./roleMapping";
import configureActivity from "./activity";
import configureMessage from "./message";
import configureBookmark from "./bookmark";
import configureTask from "./task";
import configureProject from "./project";
import configureProjectRoleMapping from "./projectRoleMapping";
import configureEnvironment from "./environment";
import configureInstance from "./instance";
import configureDatabase from "./database";
import configureDataSource from "./dataSource";

export const WORKSPACE_ID = 1;

export const OWNER_ID = 1;

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

  // message
  configureMessage(this);

  // Bookmark
  configureBookmark(this);

  // Project
  configureProject(this);

  // ProjectRoleMapping
  configureProjectRoleMapping(this);

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
