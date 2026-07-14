import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { type ReactRoute, useCurrentRoute } from "./index";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
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

vi.mock("react-router-dom", () => ({
  useLocation: () => mocks.location,
  useMatches: () => mocks.matches,
  useNavigate: () => vi.fn(),
  useParams: () => mocks.parentParams,
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useCurrentRoute", () => {
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
