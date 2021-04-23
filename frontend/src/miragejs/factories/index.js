import principal from "./principal";
import bookmark from "./bookmark";
import environment from "./environment";
import project from "./project";
import instance from "./instance";
import dataSource from "./dataSource";
import dataSourceMember from "./dataSourceMember";
import database from "./database";
import issue from "./issue";
import pipeline from "./pipeline";
import stage from "./stage";
import task from "./task";
import activity from "./activity";
import message from "./message";
import workspace from "./workspace";
import member from "./member";
import projectMember from "./projectMember";

import loginInfo from "./loginInfo";
import signupInfo from "./signupInfo";
import activateInfo from "./activateInfo";

/*
 * factories are contained in a single object, that's why we
 * destructure what's coming from users and the same should
 * be done for all future factories
 */
export default {
  ...principal,
  ...bookmark,
  ...environment,
  ...project,
  ...instance,
  ...dataSource,
  ...dataSourceMember,
  ...database,
  ...message,
  ...issue,
  ...pipeline,
  ...stage,
  ...task,
  ...activity,

  ...workspace,
  ...member,
  ...projectMember,
  ...loginInfo,
  ...signupInfo,
  ...activateInfo,
};
