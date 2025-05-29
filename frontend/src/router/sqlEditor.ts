import type { RouteRecordRaw } from "vue-router";
import SQLEditorLayout from "@/layouts/SQLEditorLayout.vue";

export const SQL_EDITOR_HOME_MODULE = "sql-editor.home";
export const SQL_EDITOR_PROJECT_MODULE = "sql-editor.project";
export const SQL_EDITOR_INSTANCE_MODULE = "sql-editor.instance";
export const SQL_EDITOR_DATABASE_MODULE = "sql-editor.database";
export const SQL_EDITOR_WORKSHEET_MODULE = "sql-editor.worksheet";
export const SQL_EDITOR_DETAIL_MODULE_LEGACY = "sql-editor.legacy-detail";
export const SQL_EDITOR_SHARE_MODULE_LEGACY = "sql-editor.legacy-share";
export const SQL_EDITOR_SETTING_MODULE = "sql-editor.setting";
export const SQL_EDITOR_SETTING_GENERAL_MODULE = "sql-editor.setting.general";
export const SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE =
  "sql-editor.setting.subscription";
export const SQL_EDITOR_SETTING_INSTANCE_MODULE = "sql-editor.setting.instance";
export const SQL_EDITOR_SETTING_ENVIRONMENT_MODULE =
  "sql-editor.setting.environment";
export const SQL_EDITOR_SETTING_PROJECT_MODULE = "sql-editor.setting.project";
export const SQL_EDITOR_SETTING_DATABASES_MODULE =
  "sql-editor.setting.databases";
export const SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE =
  "sql-editor.setting.database.detail";
export const SQL_EDITOR_SETTING_USERS_MODULE = "sql-editor.setting.users";
export const SQL_EDITOR_SETTING_MEMBERS_MODULE = "sql-editor.setting.members";
export const SQL_EDITOR_SETTING_ROLES_MODULE = "sql-editor.setting.roles";
export const SQL_EDITOR_SETTING_SSO_MODULE = "sql-editor.setting.sso";
export const SQL_EDITOR_SETTING_AUDIT_LOG_MODULE =
  "sql-editor.setting.audit-log";
export const SQL_EDITOR_SETTING_DATA_CLASSIFICATION_MODULE =
  "sql-editor.setting.data-classification";
export const SQL_EDITOR_SETTING_DATA_SEMANTIC_TYPES =
  "sql-editor.setting.semantic-types";
export const SQL_EDITOR_SETTING_GLOBAL_MASKING_MODULE =
  "sql-editor.setting.global-masking";
export const SQL_EDITOR_SETTING_PROFILE_MODULE = "sql-editor.setting.profile";

const sqlEditorRoutes: RouteRecordRaw[] = [
  {
    path: "/sql-editor",
    name: "sql-editor",
    component: SQLEditorLayout,
    children: [
      {
        path: "",
        name: SQL_EDITOR_HOME_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "projects/:project",
        name: SQL_EDITOR_PROJECT_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "projects/:project/instances/:instance/databases/:database",
        name: SQL_EDITOR_DATABASE_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "projects/:project/instances/:instance",
        name: SQL_EDITOR_INSTANCE_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "projects/:project/sheets/:sheet",
        name: SQL_EDITOR_WORKSHEET_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: ":connectionSlug",
        name: SQL_EDITOR_DETAIL_MODULE_LEGACY,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "sheet/:sheetSlug",
        name: SQL_EDITOR_SHARE_MODULE_LEGACY,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "sheet/:sheetSlug",
        name: SQL_EDITOR_SHARE_MODULE_LEGACY,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "setting",
        name: SQL_EDITOR_SETTING_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorSettingPage.vue"),
        children: [
          {
            path: "general",
            name: SQL_EDITOR_SETTING_GENERAL_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () => import("../views/sql-editor/Setting/General"),
          },
          {
            path: "subscription",
            name: SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () => import("../views/sql-editor/Setting/Subscription"),
          },
          {
            path: "instance",
            name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.instances.list"],
            },
            component: () => import("../views/sql-editor/Setting/Instance"),
          },
          {
            path: "project",
            name: SQL_EDITOR_SETTING_PROJECT_MODULE,
            component: () => import("../views/sql-editor/Setting/Project"),
          },
          {
            path: "environment",
            name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () => import("../views/sql-editor/Setting/Environment"),
          },
          {
            path: "database/instances/:instanceId/databases/:databaseName",
            name: SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.databases.list"],
            },
            component: () => import("../views/sql-editor/Setting/Database"),
          },
          {
            path: "database",
            name: SQL_EDITOR_SETTING_DATABASES_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.databases.list"],
            },
            component: () => import("../views/sql-editor/Setting/Database"),
          },
          {
            path: "users",
            name: SQL_EDITOR_SETTING_USERS_MODULE,
            component: () => import("../views/sql-editor/Setting/Users"),
          },
          {
            path: "members",
            name: SQL_EDITOR_SETTING_MEMBERS_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.policies.get"],
            },
            component: () => import("../views/sql-editor/Setting/Members"),
          },
          {
            path: "roles",
            name: SQL_EDITOR_SETTING_ROLES_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.roles.list"],
            },
            component: () => import("../views/sql-editor/Setting/Roles"),
          },
          {
            path: "sso",
            name: SQL_EDITOR_SETTING_SSO_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () => import("../views/sql-editor/Setting/SSO"),
          },
          {
            path: "audit-log",
            name: SQL_EDITOR_SETTING_AUDIT_LOG_MODULE,
            meta: {
              requiredPermissionList: () => [
                "bb.settings.get",
                "bb.auditLogs.search",
              ],
            },
            component: () => import("../views/sql-editor/Setting/AuditLog"),
          },
          {
            path: "data-classification",
            name: SQL_EDITOR_SETTING_DATA_CLASSIFICATION_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () =>
              import("../views/sql-editor/Setting/DataClassification"),
          },
          {
            path: "semantic-types",
            name: SQL_EDITOR_SETTING_DATA_SEMANTIC_TYPES,
            meta: {
              requiredPermissionList: () => ["bb.settings.get"],
            },
            component: () =>
              import("../views/sql-editor/Setting/SemanticTypes"),
          },
          {
            path: "global-masking",
            name: SQL_EDITOR_SETTING_GLOBAL_MASKING_MODULE,
            meta: {
              requiredPermissionList: () => ["bb.policies.get"],
            },
            component: () => import("../views/sql-editor/Setting/DataMasking"),
          },
          {
            path: "profile",
            name: SQL_EDITOR_SETTING_PROFILE_MODULE,
            meta: {
              requiredPermissionList: () => [],
            },
            component: () => import("../views/sql-editor/Setting/Profile"),
          },
        ],
      },
    ],
  },
];

export default sqlEditorRoutes;
