import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { EditorPanelViewState } from "@/types/sqlEditor/tabViewState";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderHook = <T,>(hookFn: () => T) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let value!: T;
  function Host() {
    value = hookFn();
    return null;
  }
  act(() => {
    root.render(<Host />);
  });
  return {
    get current() {
      return value;
    },
    unmount: () => act(() => root.unmount()),
  };
};

const mocks = vi.hoisted(() => ({
  updateTab: vi.fn(),
  currentTab: {
    id: "t1",
    viewState: {
      view: "TABLES",
      schema: "public",
      detail: { table: "users" },
    } as EditorPanelViewState,
  },
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: () => ({
    updateTab: mocks.updateTab,
    get currentTab() {
      return mocks.currentTab;
    },
  }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/types", () => ({
  defaultViewState: () => ({ view: "CODE", detail: {} }),
}));

let useViewStateNav: typeof import("./useViewStateNav").useViewStateNav;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ useViewStateNav } = await import("./useViewStateNav"));
});

describe("useViewStateNav", () => {
  test("exposes the current viewState, schema, and detail", () => {
    const result = renderHook(() => useViewStateNav());
    expect(result.current.viewState?.view).toBe("TABLES");
    expect(result.current.schema).toBe("public");
    expect(result.current.detail?.table).toBe("users");
  });

  test("updateViewState patches over the existing viewState", () => {
    const result = renderHook(() => useViewStateNav());
    result.current.updateViewState({ view: "VIEWS" });
    expect(mocks.updateTab).toHaveBeenCalledWith("t1", {
      viewState: {
        view: "VIEWS",
        schema: "public",
        detail: { table: "users" },
      },
    });
  });

  test("setDetail merges into the existing detail", () => {
    const result = renderHook(() => useViewStateNav());
    result.current.setDetail({ column: "id" });
    expect(mocks.updateTab).toHaveBeenCalledWith("t1", {
      viewState: {
        view: "TABLES",
        schema: "public",
        detail: { table: "users", column: "id" },
      },
    });
  });

  test("clearDetail resets detail to an empty object", () => {
    const result = renderHook(() => useViewStateNav());
    result.current.clearDetail();
    expect(mocks.updateTab).toHaveBeenCalledWith("t1", {
      viewState: {
        view: "TABLES",
        schema: "public",
        detail: {},
      },
    });
  });

  test("setSchema updates the schema field", () => {
    const result = renderHook(() => useViewStateNav());
    result.current.setSchema("inventory");
    expect(mocks.updateTab).toHaveBeenCalledWith("t1", {
      viewState: {
        view: "TABLES",
        schema: "inventory",
        detail: { table: "users" },
      },
    });
  });
});
