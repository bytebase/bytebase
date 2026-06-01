import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Pinia bridge — runs the getter against the (mocked) Pinia stores.
  usePiniaBridge: vi.fn<(getter: () => unknown) => unknown>(),
  // Zustand editor store project name.
  project: "projects/test" as string,
  // useAppProject (app store) return value.
  projectData: { allowJustInTimeAccess: false } as {
    allowJustInTimeAccess: boolean;
  },
  // New zustand store.
  state: {
    asidePanelTab: "WORKSHEET" as string,
  },
  setAsidePanelTab: vi.fn(),
  routerResolve: vi.fn(() => ({ href: "/project/test-project" })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/usePiniaBridge", () => ({
  usePiniaBridge: mocks.usePiniaBridge,
}));

vi.mock("@/react/hooks/useAppProject", () => ({
  useAppProject: () => mocks.projectData,
}));

vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: {
      asidePanelTab: string;
      setAsidePanelTab: (tab: string) => void;
    }) => unknown
  ) =>
    selector({
      asidePanelTab: mocks.state.asidePanelTab,
      setAsidePanelTab: mocks.setAsidePanelTab,
    }),
}));

vi.mock("@/router", () => ({
  router: {
    resolve: mocks.routerResolve,
    currentRoute: { value: { params: {} } },
    afterEach: () => () => {},
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DETAIL: "project.detail",
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
}));

vi.mock("@/assets/logo-icon.svg", () => ({
  default: "/assets/logo-icon.svg",
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let GutterBar: typeof import("./GutterBar").GutterBar;

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
  mocks.project = "projects/test";
  mocks.projectData = { allowJustInTimeAccess: false };
  mocks.state.asidePanelTab = "WORKSHEET";
  mocks.routerResolve.mockReturnValue({ href: "/workspace/home" });
  ({ GutterBar } = await import("./GutterBar"));
});

describe("GutterBar", () => {
  test("renders 3 tabs when project does not allow JIT access", () => {
    mocks.usePiniaBridge.mockImplementation((getter) => getter());
    mocks.projectData = { allowJustInTimeAccess: false };
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    expect(buttons).toHaveLength(3);
    unmount();
  });

  test("renders 4 tabs when project allows JIT access", () => {
    mocks.usePiniaBridge.mockImplementation((getter) => getter());
    mocks.projectData = { allowJustInTimeAccess: true };
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    expect(buttons).toHaveLength(4);
    unmount();
  });

  test("click writes asidePanelTab via setAsidePanelTab", () => {
    mocks.usePiniaBridge.mockImplementation((getter) => getter());
    mocks.projectData = { allowJustInTimeAccess: false };
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const buttons = container.querySelectorAll("button");
    act(() => {
      (buttons[1] as HTMLButtonElement).click();
    });
    expect(mocks.setAsidePanelTab).toHaveBeenCalledWith("SCHEMA");
    unmount();
  });

  test("logo link has target=_blank and rel=noopener noreferrer", () => {
    mocks.usePiniaBridge.mockImplementation((getter) => getter());
    mocks.projectData = { allowJustInTimeAccess: false };
    mocks.routerResolve.mockReturnValue({ href: "/workspace/home" });
    const { container, render, unmount } = renderIntoContainer(<GutterBar />);
    render();
    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toBe("/workspace/home");
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toBe("noopener noreferrer");
    unmount();
  });
});
