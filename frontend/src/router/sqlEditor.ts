import { defineComponent, h } from "vue";
import type { RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import ReactPageMount from "@/react/ReactPageMount.vue";

export const SQL_EDITOR_HOME_MODULE = "sql-editor.home";
export const SQL_EDITOR_PROJECT_MODULE = "sql-editor.project";
export const SQL_EDITOR_INSTANCE_MODULE = "sql-editor.instance";
export const SQL_EDITOR_DATABASE_MODULE = "sql-editor.database";
export const SQL_EDITOR_WORKSHEET_MODULE = "sql-editor.worksheet";

// Parent route component: a one-line Vue render-only shell that mounts
// the React `SQLEditorLayout` tree. The full layout (debug teleport
// target, banners wrapper, route shell, home page, AI bridge, etc.)
// lives React-side; this shim exists only because Vue Router still
// owns the route entry. Migrating the route itself to React Router is
// out of scope per Stage 21 design § 4.8 option A.
const SQLEditorLayoutComponent = defineComponent({
  name: "SQLEditorLayoutRoute",
  render: () =>
    h(ReactPageMount, {
      page: "SQLEditorLayout",
      containerClass: "w-full h-full flex flex-col",
    }),
});

// Child routes don't render anything of their own — the React layout
// inspects `useCurrentRoute()` and decides what to show. We only need
// each child to exist so `router.push({ name: SQL_EDITOR_*_MODULE })`
// resolves to a valid path. Vue Router requires a `component` field;
// rendering nothing is the closest match to "no body".
const NoopRouteComponent = defineComponent({
  name: "SQLEditorChildRoute",
  render: () => null,
});

const sqlEditorRoutes: RouteRecordRaw[] = [
  {
    path: "/sql-editor",
    name: "sql-editor",
    component: SQLEditorLayoutComponent,
    meta: {
      requiredPermissionList: () => [
        "bb.projects.get",
        "bb.databases.list",
        "bb.projects.getIamPolicy",
      ],
    },
    children: [
      {
        path: "",
        name: SQL_EDITOR_HOME_MODULE,
        meta: { title: () => t("sql-editor.self") },
        component: NoopRouteComponent,
      },
      {
        path: "projects/:project",
        name: SQL_EDITOR_PROJECT_MODULE,
        meta: { title: () => t("sql-editor.self") },
        component: NoopRouteComponent,
      },
      {
        path: "projects/:project/instances/:instance/databases/:database",
        name: SQL_EDITOR_DATABASE_MODULE,
        meta: { title: () => t("sql-editor.self") },
        component: NoopRouteComponent,
      },
      {
        path: "projects/:project/instances/:instance",
        name: SQL_EDITOR_INSTANCE_MODULE,
        meta: { title: () => t("sql-editor.self") },
        component: NoopRouteComponent,
      },
      {
        path: "projects/:project/sheets/:sheet",
        name: SQL_EDITOR_WORKSHEET_MODULE,
        meta: { title: () => t("sql-editor.self") },
        component: NoopRouteComponent,
      },
    ],
  },
];

export default sqlEditorRoutes;
