import { lazy, Suspense } from "react";
import { Outlet, type RouteObject } from "react-router-dom";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  SQL_EDITOR_QUERY_HISTORY_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
} from "@/react/router/handles";
import type { Permission } from "@/types";

// `SQLEditorLayout` is a "layout as a page": the React component inspects the
// current route itself and decides what to render. It does NOT render an
// `<Outlet/>`, so we wrap it in a small layout-route component that renders
// the layout alongside an `<Outlet/>`. The child routes carry no element of
// their own — they exist only so navigation by route name resolves to a path,
// matching the original vue `NoopRouteComponent` children.
const SQLEditorLayout = lazy(() =>
  import("@/react/components/sql-editor/SQLEditorLayout").then((m) => ({
    default: m.SQLEditorLayout,
  }))
);

const SqlEditorLayoutRoute = () => (
  <Suspense fallback={null}>
    <SQLEditorLayout />
    <Outlet />
  </Suspense>
);

export const sqlEditorRoutes: RouteObject[] = [
  {
    path: "/sql-editor",
    // `layoutAsPage` marks this parent as rendering its own content (the
    // `SQLEditorLayout` inspects the route and draws everything); its child
    // routes are intentionally element-less and render into an empty Outlet.
    // The route-reachability test relies on this flag to allow those children.
    // `SQLEditorRouteShell` gates the editor on `route.requiredPermissions`,
    // so the parent must carry the route permission list (ported 1:1 from the
    // legacy vue `/sql-editor` route). The child modules inherit it via the
    // matched-chain aggregation in `assembleRoute`.
    handle: {
      name: "sql-editor",
      layoutAsPage: true,
      requiredPermissionList: (): Permission[] => [
        "bb.projects.get",
        "bb.databases.list",
        "bb.projects.getIamPolicy",
      ],
    },
    element: <SqlEditorLayoutRoute />,
    children: [
      {
        index: true,
        handle: { name: SQL_EDITOR_HOME_MODULE },
      },
      {
        path: "projects/:project",
        handle: { name: SQL_EDITOR_PROJECT_MODULE },
      },
      {
        path: "projects/:project/instances/:instance/databases/:database",
        handle: { name: SQL_EDITOR_DATABASE_MODULE },
      },
      {
        path: "projects/:project/instances/:instance",
        handle: { name: SQL_EDITOR_INSTANCE_MODULE },
      },
      {
        path: "projects/:project/sheets/:sheet",
        handle: { name: SQL_EDITOR_WORKSHEET_MODULE },
      },
      {
        path: "projects/:project/queryHistories/:queryHistory",
        handle: { name: SQL_EDITOR_QUERY_HISTORY_MODULE },
      },
    ],
  },
];
