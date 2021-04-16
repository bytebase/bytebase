/*
 * Mirage JS guide on Models: https://miragejs.com/docs/data-layer/models
 */

import { Model, hasMany, belongsTo } from "miragejs";

/*
 * Everytime you create a new resource you have
 * to create a new Model and add it here. It is
 * true for Factories and for Fixtures.
 *
 * Mirage JS guide on Relationships: https://miragejs.com/docs/main-concepts/relationships/
 */
export default {
  user: Model.extend({}),

  userNew: Model,

  userPatch: Model,

  workspace: Model.extend({
    member: hasMany(),
    bookmark: hasMany(),
    activity: hasMany(),
    message: hasMany(),
    project: hasMany(),
    projectMember: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
    database: hasMany(),
    dataSource: hasMany(),
  }),

  member: Model.extend({
    workspace: belongsTo(),
  }),

  projectMember: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
  }),

  projectMemberNew: Model,

  projectMemberPatch: Model,

  bookmark: Model.extend({
    workspace: belongsTo(),
  }),

  activity: Model.extend({
    workspace: belongsTo(),
  }),

  activityPatch: Model,

  message: Model.extend({
    workspace: belongsTo(),
  }),

  messagePatch: Model,

  project: Model.extend({
    workspace: belongsTo(),
    database: hasMany(),
    projectMember: hasMany(),
    task: hasMany(),
  }),

  projectNew: Model,

  projectPatch: Model,

  task: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
  }),

  taskNew: Model,

  taskPatch: Model,

  environment: Model.extend({
    workspace: belongsTo(),
    instance: hasMany(),
  }),

  environmentNew: Model,

  environmentPatch: Model,

  instance: Model.extend({
    workspace: belongsTo(),
    environment: belongsTo(),
    dataSource: hasMany(),
    database: hasMany(),
  }),

  instanceNew: Model,

  instancePatch: Model,

  database: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    project: belongsTo(),
    dataSource: hasMany(),
  }),

  databasePatch: Model,

  dataSource: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    database: belongsTo(),
    dataSourceMember: hasMany(),
  }),

  dataSourceMember: Model,

  dataSourcePatch: Model,

  batchUpdate: Model,

  signupInfo: Model,

  loginInfo: Model,

  activateInfo: Model,
};
