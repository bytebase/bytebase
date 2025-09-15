import type { RouteRecordRaw } from "vue-router";
import SQLEditorLayout from "@/layouts/SQLEditorLayout.vue";

export const SQL_EDITOR_HOME_MODULE = "sql-editor.home";
export const SQL_EDITOR_PROJECT_MODULE = "sql-editor.project";
export const SQL_EDITOR_INSTANCE_MODULE = "sql-editor.instance";
export const SQL_EDITOR_DATABASE_MODULE = "sql-editor.database";
export const SQL_EDITOR_WORKSHEET_MODULE = "sql-editor.worksheet";
export const SQL_EDITOR_DETAIL_MODULE_LEGACY = "sql-editor.legacy-detail";
export const SQL_EDITOR_SHARE_MODULE_LEGACY = "sql-editor.legacy-share";

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
    ],
  },
];

export default sqlEditorRoutes;
