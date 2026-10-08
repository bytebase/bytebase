import { act } from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, Outlet } from "react-router";
import { RouterProvider } from "react-router/dom";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
} from "./handles";
import {
  type ReactRoute,
  resolveRouteName,
  router,
  useCurrentRoute,
} from "./index";
import { setAppRouter, setRouteNameIndex } from "./navigation";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  // When true, the router hooks return the fixture below instead of reading a
  // router context; otherwise they are the real react-router hooks.
  useFixtureRoute: false,
  location: {
    pathname: "/sql-editor/projects/proj1/instances/inst1/databases/db1",
    search: "?schema=public",
    hash: "",
  },
  parentParams: {},
  matches: [
    {
      handle: { name: "sql-editor" },
      params: {},
    },
    {
      handle: { name: "sql-editor.database" },
      params: {
        project: "proj1",
        instance: "inst1",
        database: "db1",
      },
    },
  ],
}));

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    useLocation: () =>
      mocks.useFixtureRoute ? mocks.location : actual.useLocation(),
    useMatches: () =>
      mocks.useFixtureRoute ? mocks.matches : actual.useMatches(),
    useParams: () =>
      mocks.useFixtureRoute ? mocks.parentParams : actual.useParams(),
  };
});

const registerRouterOnWebhooksPage = () => {
  setRouteNameIndex(
    new Map<string, string>([
      [PROJECT_V1_ROUTE_WEBHOOKS, "/projects/:projectId/webhooks"],
      [PROJECT_V1_ROUTE_WEBHOOK_CREATE, "/projects/:projectId/webhooks/new"],
      [
        PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
        "/projects/:projectId/webhooks/:webhookResourceId",
      ],
    ])
  );
  setAppRouter({
    navigate: vi.fn(),
    state: {
      location: {
        pathname: "/projects/project-sample/webhooks",
        search: "",
        hash: "",
      },
      matches: [
        {
          handle: { name: PROJECT_V1_ROUTE_WEBHOOKS },
          params: { projectId: "project-sample" },
        },
      ],
      initialized: true,
    },
  });
};

describe("useCurrentRoute", () => {
  beforeEach(() => {
    mocks.useFixtureRoute = true;
  });

  afterEach(() => {
    mocks.useFixtureRoute = false;
  });

  test("uses leaf match params when rendered from a parent layout", () => {
    let route: ReactRoute | undefined;
    const CaptureRoute = () => {
      route = useCurrentRoute();
      return null;
    };

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(<CaptureRoute />);
    });

    expect(route?.name).toBe("sql-editor.database");
    expect(route?.params).toEqual({
      project: "proj1",
      instance: "inst1",
      database: "db1",
    });

    act(() => {
      root.unmount();
      container.remove();
    });
  });
});

describe("useCurrentRoute with real router context", () => {
  test("returns leaf SQL Editor params when called from the parent layout", async () => {
    let route: ReactRoute | undefined;
    const CaptureFromParentLayout = () => {
      route = useCurrentRoute();
      return <Outlet />;
    };

    const memoryRouter = createMemoryRouter(
      [
        {
          path: "/sql-editor",
          handle: { name: "sql-editor" },
          element: <CaptureFromParentLayout />,
          children: [
            {
              path: "projects/:project/instances/:instance/databases/:database",
              handle: { name: SQL_EDITOR_DATABASE_MODULE },
              element: null,
            },
          ],
        },
      ],
      {
        initialEntries: [
          "/sql-editor/projects/proj1/instances/inst1/databases/db1?schema=public",
        ],
      }
    );

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(<RouterProvider router={memoryRouter} />);
    });

    expect(route?.name).toBe(SQL_EDITOR_DATABASE_MODULE);
    expect(route?.params).toEqual({
      project: "proj1",
      instance: "inst1",
      database: "db1",
    });
    expect(route?.query).toEqual({ schema: "public" });

    act(() => {
      root.unmount();
      container.remove();
    });
  });
});

// Guardrail for the "getSnapshot should be cached" infinite-loop class.
// `router.currentRoute.value` backs the `useReactiveRoute` `useSyncExternalStore`
// snapshot. If it returns a fresh object on every read, any external-store
// consumer re-renders forever (this crashed the SQL Editor: GutterBar →
// useReactiveRoute → "Maximum update depth exceeded"). The snapshot is memoized in
// `currentRouteSnapshot` to keep a stable reference between actual route
// changes; this test fails if that memoization regresses.
describe("router.currentRoute snapshot stability", () => {
  beforeEach(registerRouterOnWebhooksPage);

  test("returns a referentially stable object across reads without navigation", () => {
    const a = router.currentRoute.value;
    const b = router.currentRoute.value;
    expect(a).toBe(b);
  });
});

describe("router named route params", () => {
  beforeEach(registerRouterOnWebhooksPage);

  test("inherits current params when resolving project child routes", () => {
    expect(
      router.resolve({ name: PROJECT_V1_ROUTE_WEBHOOK_CREATE }).fullPath
    ).toBe("/projects/project-sample/webhooks/new");
    expect(
      router.resolve({
        name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
        params: { webhookResourceId: "hook-1" },
      }).fullPath
    ).toBe("/projects/project-sample/webhooks/hook-1");
  });

  test("resolves a pathname to the most specific registered route name", () => {
    expect(resolveRouteName("/projects/project-sample/webhooks/new")).toBe(
      PROJECT_V1_ROUTE_WEBHOOK_CREATE
    );
    expect(resolveRouteName("/projects/project-sample/webhooks/hook-1")).toBe(
      PROJECT_V1_ROUTE_WEBHOOK_DETAIL
    );
  });

  test("prefers the leaf route name when parent and index routes share a path", () => {
    setRouteNameIndex(
      new Map<string, string>([
        ["sql-editor", "/sql-editor"],
        [SQL_EDITOR_HOME_MODULE, "/sql-editor"],
      ])
    );

    expect(resolveRouteName("/sql-editor")).toBe(SQL_EDITOR_HOME_MODULE);
  });
});
