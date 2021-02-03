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
  user: Model.extend({
    workspaceRole: hasMany(),
  }),

  workspace: Model.extend({
    group: hasMany(),
    project: hasMany(),
    workspaceRole: hasMany(),
    pipeline: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
  }),

  workspaceRole: Model.extend({
    workspace: belongsTo(),
    user: belongsTo(),
  }),

  bookmark: Model.extend({
    workspace: belongsTo(),
  }),

  activity: Model.extend({
    workspace: belongsTo(),
  }),

  group: Model.extend({
    workspace: belongsTo(),
    project: hasMany(),
    groupRole: hasMany(),
  }),

  groupRole: Model.extend({
    group: belongsTo(),
    user: belongsTo(),
  }),

  pipeline: Model.extend({
    workspace: belongsTo(),
  }),

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

  project: Model.extend({
    workspace: belongsTo(),
    group: belongsTo(),
    environment: hasMany(),
    // To signal 1:1 relationship
    repository: belongsTo(),
  }),

  repository: Model.extend({
    project: belongsTo(),
  }),

  job: Model.extend({
    step: hasMany(),
  }),

  step: Model.extend({
    job: belongsTo(),
  }),

  sortOrder: Model,

  signupInfo: Model,

  loginInfo: Model,
};
