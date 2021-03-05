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
    membership: hasMany(),
    activity: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
  }),

  membership: Model.extend({
    workspace: belongsTo(),
  }),

  bookmark: Model.extend({
    workspace: belongsTo(),
  }),

  activity: Model.extend({
    workspace: belongsTo(),
  }),

  activityPatch: Model,

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

  batchUpdate: Model,

  signupInfo: Model,

  loginInfo: Model,
};
