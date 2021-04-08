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

  workspace: Model.extend({
    roleMapping: hasMany(),
    activity: hasMany(),
    message: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
  }),

  roleMapping: Model.extend({
    workspace: belongsTo(),
  }),

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

  task: Model.extend({
    workspace: belongsTo(),
  }),

  taskPatch: Model,

  environment: Model.extend({
    workspace: belongsTo(),
    instance: hasMany(),
  }),

  instance: Model.extend({
    workspace: belongsTo(),
    environment: belongsTo(),
    dataSource: hasMany(),
    database: hasMany(),
  }),

  database: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
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
