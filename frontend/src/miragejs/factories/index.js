import user from "./user";
import bookmark from "./bookmark";
import environment from "./environment";
import project from "./project";
import instance from "./instance";
import dataSource from "./dataSource";
import dataSourceMember from "./dataSourceMember";
import database from "./database";
import task from "./task";
import stage from "./stage";
import step from "./step";
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
  ...user,
  ...bookmark,
  ...environment,
  ...project,
  ...instance,
  ...dataSource,
  ...dataSourceMember,
  ...database,
  ...message,
  ...task,
  ...stage,
  ...step,
  ...activity,

  ...workspace,
  ...member,
  ...projectMember,
  ...loginInfo,
  ...signupInfo,
  ...activateInfo,
};
