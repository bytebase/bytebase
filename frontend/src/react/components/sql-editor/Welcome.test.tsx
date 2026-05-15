import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  usePermissionCheck:
    vi.fn<
      (
        perms: readonly string[],
        project?: unknown
      ) => [boolean, string | undefined]
    >(),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  routerPush: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  usePermissionCheck: mocks.usePermissionCheck,
}));

vi.mock("@/react/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  INSTANCE_ROUTE_DASHBOARD: "workspace.instance",
  PROJECT_V1_ROUTE_DASHBOARD: "workspace.project",
}));

vi.mock("@/store", () => ({
  useProjectV1Store: vi.fn(),
  useWorkspaceV1Store: vi.fn(),
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: vi.fn(),
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

let Welcome: typeof import("./Welcome").Welcome;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  // Default: workspace logo empty, both permissions granted.
  mocks.useVueState.mockReturnValue("");
  mocks.usePermissionCheck.mockReturnValue([true, undefined]);
  ({ Welcome } = await import("./Welcome"));
});

describe("Welcome", () => {
  test("renders both buttons when both permissions present", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.add-a-new-instance");
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    unmount();
  });

  test("hides Add-Instance when missing bb.instances.create", () => {
    mocks.usePermissionCheck.mockImplementation((perms) => {
      if (perms.includes("bb.instances.create")) return [false, "missing"];
      return [true, undefined];
    });
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).not.toContain(
      "sql-editor.add-a-new-instance"
    );
    expect(container.textContent).toContain("sql-editor.connect-to-a-database");
    unmount();
  });

  test("hides Connect when missing bb.sql.select", () => {
    mocks.usePermissionCheck.mockImplementation((perms) => {
      if (perms.includes("bb.sql.select")) return [false, "missing"];
      return [true, undefined];
    });
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.textContent).toContain("sql-editor.add-a-new-instance");
    expect(container.textContent).not.toContain(
      "sql-editor.connect-to-a-database"
    );
    unmount();
  });

  test("hides both buttons when neither permission present", () => {
    mocks.usePermissionCheck.mockReturnValue([false, "missing"]);
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(container.querySelectorAll("button")).toHaveLength(0);
    unmount();
  });

  test("routes to instance dashboard with #add hash on Add-Instance click", () => {
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    const buttons = container.querySelectorAll("button");
    // Add-Instance is the first button (matches Vue order).
    act(() => {
      (buttons[0] as HTMLButtonElement).click();
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
      hash: "#add",
    });
    unmount();
  });

  test("invokes onChangeConnection on Connect click", () => {
    const onChangeConnection = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={onChangeConnection} />
    );
    render();
    const buttons = container.querySelectorAll("button");
    // Connect is the second button (matches Vue order).
    act(() => {
      (buttons[1] as HTMLButtonElement).click();
    });
    expect(onChangeConnection).toHaveBeenCalledTimes(1);
    unmount();
  });
});
