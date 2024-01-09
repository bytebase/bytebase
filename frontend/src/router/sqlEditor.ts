import { RouteRecordRaw } from "vue-router";
import SQLEditorLayout from "@/layouts/SQLEditorLayout.vue";

export const SQL_EDITOR_HOME_MODULE = "sql-editor.home";
export const SQL_EDITOR_DETAIL_MODULE = "sql-editor.detail";
export const SQL_EDITOR_SHARE_MODULE = "sql-editor.share";

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
        path: ":connectionSlug",
        name: SQL_EDITOR_DETAIL_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
      {
        path: "sheet/:sheetSlug",
        name: SQL_EDITOR_SHARE_MODULE,
        meta: { title: () => "Bytebase SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
      },
    ],
  },
];

export default sqlEditorRoutes;
