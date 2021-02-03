/*
 * Mirage JS guide on Factories: https://miragejs.com/docs/data-layer/factories
 */

import activity from "./activity";
import user from "./user";
import bookmark from "./bookmark";
import project from "./project";
import environment from "./environment";
import instance from "./instance";
import dataSource from "./dataSource";
import repository from "./repository";
import job from "./job";
import step from "./step";
import pipeline from "./pipeline";
import sortOrder from "./sortOrder";
import group from "./group";
import workspace from "./workspace";
import workspaceRole from "./workspaceRole";

import loginInfo from "./loginInfo";

/*
 * factories are contained in a single object, that's why we
 * destructure what's coming from users and the same should
 * be done for all future factories
 */
export default {
  ...activity,
  ...user,
  ...bookmark,
  ...project,
  ...environment,
  ...instance,
  ...dataSource,
  ...repository,
  ...job,
  ...step,
  ...pipeline,
  ...sortOrder,
  ...group,
  ...workspace,
  ...workspaceRole,
  ...loginInfo,
};
