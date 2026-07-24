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
  routerPush: vi.fn(),
  projectData: { name: "projects/test" } as { name: string },
  themeDark: false,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/components/PermissionGuard", () => ({
  usePermissionCheck: mocks.usePermissionCheck,
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: ({ builtinTheme }: { builtinTheme?: string }) => (
    <div data-testid="welcome-logo" data-builtin-theme={builtinTheme} />
  ),
}));

vi.mock("@/hooks/useAppProject", () => ({
  useAppProject: () => mocks.projectData,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: { push: mocks.routerPush },
}));

vi.mock("@/stores", () => ({
  useProjectV1Store: vi.fn(),
}));

vi.mock("@/modules/sql-editor/store/editor-vue-state", () => ({
  useSQLEditorVueState: vi.fn(),
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

vi.mock("@/modules/sql-editor/components/theme/SQLEditorThemeScope", () => ({
  useSQLEditorTheme: () => ({
    id: mocks.themeDark ? "dark" : "light",
  }),
}));

vi.mock("@/modules/sql-editor/components/theme/derive", () => ({
  isDarkTheme: () => mocks.themeDark,
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
  mocks.themeDark = false;
  // Default: both permissions granted.
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

  test("passes dark SQL Editor theme to the logo", () => {
    mocks.themeDark = true;
    const { container, render, unmount } = renderIntoContainer(
      <Welcome onChangeConnection={() => {}} />
    );
    render();
    expect(
      container.querySelector('[data-testid="welcome-logo"]')?.getAttribute(
        "data-builtin-theme"
      )
    ).toBe("dark");
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
