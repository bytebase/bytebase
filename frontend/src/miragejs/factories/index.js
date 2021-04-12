import user from "./user";
import bookmark from "./bookmark";
import environment from "./environment";
import project from "./project";
import instance from "./instance";
import dataSource from "./dataSource";
import dataSourceMember from "./dataSourceMember";
import database from "./database";
import task from "./task";
import activity from "./activity";
import message from "./message";
import workspace from "./workspace";
import roleMapping from "./roleMapping";

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
  ...activity,

  ...workspace,
  ...roleMapping,
  ...loginInfo,
  ...signupInfo,
  ...activateInfo,
};
