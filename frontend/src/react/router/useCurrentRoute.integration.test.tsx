import { act } from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";
import { describe, expect, test } from "vitest";
import { SQL_EDITOR_DATABASE_MODULE } from "./handles";
import { useCurrentRoute, type ReactRoute } from "./index";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("useCurrentRoute with real router context", () => {
  test("returns leaf SQL Editor params when called from the parent layout", async () => {
    let route: ReactRoute | undefined;
    const CaptureFromParentLayout = () => {
      route = useCurrentRoute();
      return <Outlet />;
    };

    const router = createMemoryRouter(
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
      root.render(<RouterProvider router={router} />);
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
