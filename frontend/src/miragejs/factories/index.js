/*
 * Mirage JS guide on Factories: https://miragejs.com/docs/data-layer/factories
 */

import user from "./user";
import bookmark from "./bookmark";
import project from "./project";
import environment from "./environment";
import instance from "./instance";
import dataSource from "./dataSource";
import repository from "./repository";
import task from "./task";
import taskPatch from "./taskPatch";
import activity from "./activity";
import activityPatch from "./activityPatch";
import group from "./group";
import workspace from "./workspace";
import member from "./member";

import batchUpdate from "./batchUpdate";
import loginInfo from "./loginInfo";

/*
 * factories are contained in a single object, that's why we
 * destructure what's coming from users and the same should
 * be done for all future factories
 */
export default {
  ...user,
  ...bookmark,
  ...project,
  ...environment,
  ...instance,
  ...dataSource,
  ...repository,
  ...task,
  ...taskPatch,
  ...activity,
  ...activityPatch,
  ...group,
  ...workspace,
  ...member,
  ...batchUpdate,
  ...loginInfo,
};
