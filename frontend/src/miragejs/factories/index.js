import user from "./user";
import bookmark from "./bookmark";
import environment from "./environment";
import instance from "./instance";
import dataSource from "./dataSource";
import dataSourceMember from "./dataSourceMember";
import database from "./database";
import databasePatch from "./databasePatch";
import task from "./task";
import taskPatch from "./taskPatch";
import activity from "./activity";
import activityPatch from "./activityPatch";
import message from "./message";
import messagePatch from "./messagePatch";
import workspace from "./workspace";
import roleMapping from "./roleMapping";

import batchUpdate from "./batchUpdate";
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
  ...instance,
  ...dataSource,
  ...dataSourceMember,
  ...database,
  ...databasePatch,
  ...message,
  ...messagePatch,
  ...task,
  ...taskPatch,
  ...activity,
  ...activityPatch,

  ...workspace,
  ...roleMapping,
  ...batchUpdate,
  ...loginInfo,
  ...signupInfo,
  ...activateInfo,
};
