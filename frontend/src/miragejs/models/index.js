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
    roleMapping: hasMany(),
    bookmark: hasMany(),
    activity: hasMany(),
    message: hasMany(),
    project: hasMany(),
    projectRoleMapping: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
    database: hasMany(),
    dataSource: hasMany(),
  }),

  roleMapping: Model.extend({
    workspace: belongsTo(),
  }),

  projectRoleMapping: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
  }),

  projectRoleMappingNew: Model,

  projectRoleMappingPatch: Model,

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
    projectRoleMapping: hasMany(),
  }),

  projectNew: Model,

  projectPatch: Model,

  task: Model.extend({
    workspace: belongsTo(),
  }),

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

  batchUpdate: Model,

  signupInfo: Model,

  loginInfo: Model,

  activateInfo: Model,
};
