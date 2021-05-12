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
  // Principal
  principal: Model.extend({}),
  principalCreate: Model,
  principalPatch: Model,

  // Workspace
  workspace: Model.extend({
    member: hasMany(),
    bookmark: hasMany(),
    activity: hasMany(),
    message: hasMany(),
    project: hasMany(),
    projectMember: hasMany(),
    issue: hasMany(),
    pipeline: hasMany(),
    stage: hasMany(),
    task: hasMany(),
    environment: hasMany(),
    instance: hasMany(),
    database: hasMany(),
    dataSource: hasMany(),
  }),

  // Member
  member: Model.extend({
    workspace: belongsTo(),
  }),
  memberCreate: Model,
  memberPatch: Model,

  // Bookmark
  bookmark: Model.extend({
    workspace: belongsTo(),
  }),
  bookmarkCreate: Model,

  // Activity
  activity: Model.extend({
    workspace: belongsTo(),
  }),
  activityCreate: Model,
  activityPatch: Model,

  // Message
  message: Model.extend({
    workspace: belongsTo(),
  }),
  messagePatch: Model,

  // Project
  project: Model.extend({
    workspace: belongsTo(),
    database: hasMany(),
    projectMember: hasMany(),
    issue: hasMany(),
  }),
  projectCreate: Model,
  projectPatch: Model,

  // Project Member
  projectMember: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
  }),
  projectMemberCreate: Model,
  projectMemberPatch: Model,

  // Issue
  issue: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
    pipeline: belongsTo(),
  }),
  issueCreate: Model,
  issuePatch: Model,
  issueStatusPatch: Model,

  // Pipeline
  pipeline: Model.extend({
    workspace: belongsTo(),
    stage: hasMany(),
    task: hasMany(),
  }),
  pipelinePatch: Model,
  pipelineStatusPatch: Model,

  // Stage
  stage: Model.extend({
    workspace: belongsTo(),
    pipeline: belongsTo(),
    environment: belongsTo(),
  }),
  stagePatch: Model,
  stageStatusPatch: Model,

  // Task
  task: Model.extend({
    workspace: belongsTo(),
    pipeline: belongsTo(),
    stage: belongsTo(),
    database: belongsTo(),
  }),
  taskPatch: Model,
  taskStatusPatch: Model,

  // Environment
  environment: Model.extend({
    workspace: belongsTo(),
    instance: hasMany(),
  }),
  environmentCreate: Model,
  environmentPatch: Model,

  // Instance
  instance: Model.extend({
    workspace: belongsTo(),
    environment: belongsTo(),
    dataSource: hasMany(),
    database: hasMany(),
  }),
  instanceCreate: Model,
  instancePatch: Model,

  // Database
  database: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    project: belongsTo(),
    dataSource: hasMany(),
    task: hasMany(),
  }),
  databaseCreate: Model,
  databasePatch: Model,

  // Data Source
  dataSource: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    database: belongsTo(),
    dataSourceMember: hasMany(),
  }),
  dataSourceCreate: Model,
  dataSourcePatch: Model,

  // Data Source Member
  dataSourceMember: Model,
  dataSourceMemberCreate: Model,

  // Misc
  signupInfo: Model,

  loginInfo: Model,

  activateInfo: Model,
};
