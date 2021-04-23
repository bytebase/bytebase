/*
 * Mirage JS guide on Routes: https://miragejs.com/docs/route-handlers/functions
 */
import configurePrincipal from "./principal";
import configureAuth from "./auth";
import configureMember from "./member";
import configureActivity from "./activity";
import configureMessage from "./message";
import configureBookmark from "./bookmark";
import configureIssue from "./issue";
import configureTask from "./task";
import configureProject from "./project";
import configureProjectMember from "./projectMember";
import configureEnvironment from "./environment";
import configureInstance from "./instance";
import configureDatabase from "./database";
import configureDataSource from "./dataSource";

export const WORKSPACE_ID = 1;

// TODO: Use the OWNER ID for now.
// In actual implementation, we need to fetch the user from the auth context.
export const FAKE_API_CALLER_ID = 1;

export default function routes() {
  // Change this value to simulate response delay.
  // By default development environment has a 400ms delay.
  this.timing = 0;

  this.namespace = "api";

  // Principal
  configurePrincipal(this);

  // Auth
  configureAuth(this);

  // Member
  configureMember(this);

  // Activity
  configureActivity(this);

  // message
  configureMessage(this);

  // Bookmark
  configureBookmark(this);

  // Project
  configureProject(this);

  // ProjectMember
  configureProjectMember(this);

  // Issue
  configureIssue(this);

  // Task
  configureTask(this);

  // Environment
  configureEnvironment(this);

  // Instance
  configureInstance(this);

  // Database
  configureDatabase(this);

  // Data Source
  // Disable data source related route for now
  // as we only allow interacting with admin data source via instance API
  // configureDataSource(this);
}
