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
    group: hasMany(),
    member: hasMany(),
    activity: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
  }),

  member: Model.extend({
    workspace: belongsTo(),
  }),

  bookmark: Model.extend({
    workspace: belongsTo(),
  }),

  activity: Model.extend({
    workspace: belongsTo(),
  }),

  activityPatch: Model,

  group: Model.extend({
    workspace: belongsTo(),

    groupRole: hasMany(),
  }),

  groupRole: Model.extend({
    group: belongsTo(),
    user: belongsTo(),
  }),

  task: Model.extend({
    workspace: belongsTo(),
  }),

  taskPatch: Model,

  environment: Model.extend({
    workspace: belongsTo(),
  }),

  instance: Model.extend({
    workspace: belongsTo(),
    dataSource: hasMany(),
  }),

  dataSource: Model.extend({
    instance: belongsTo(),
  }),

  repository: Model.extend({}),

  batchUpdate: Model,

  signupInfo: Model,

  loginInfo: Model,
};
