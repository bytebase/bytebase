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
  // User
  user: Model.extend({}),
  userNew: Model,
  userPatch: Model,

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
  memberNew: Model,
  memberPatch: Model,

  // Bookmark
  bookmark: Model.extend({
    workspace: belongsTo(),
  }),
  bookmarkNew: Model,

  // Activity
  activity: Model.extend({
    workspace: belongsTo(),
  }),
  activityNew: Model,
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
  projectNew: Model,
  projectPatch: Model,

  // Project Member
  projectMember: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
  }),
  projectMemberNew: Model,
  projectMemberPatch: Model,

  // Issue
  issue: Model.extend({
    workspace: belongsTo(),
    project: belongsTo(),
    pipeline: belongsTo(),
  }),
  issueNew: Model,
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
  environmentNew: Model,
  environmentPatch: Model,

  // Instance
  instance: Model.extend({
    workspace: belongsTo(),
    environment: belongsTo(),
    dataSource: hasMany(),
    database: hasMany(),
  }),
  instanceNew: Model,
  instancePatch: Model,

  // Database
  database: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    project: belongsTo(),
    dataSource: hasMany(),
    task: hasMany(),
  }),
  databaseNew: Model,
  databasePatch: Model,

  // Data Source
  dataSource: Model.extend({
    workspace: belongsTo(),
    instance: belongsTo(),
    database: belongsTo(),
    dataSourceMember: hasMany(),
  }),
  dataSourceNew: Model,
  dataSourcePatch: Model,

  // Data Source Member
  dataSourceMember: Model,
  dataSourceMemberNew: Model,

  // Misc
  batchUpdate: Model,

  signupInfo: Model,

  loginInfo: Model,

  activateInfo: Model,
};
