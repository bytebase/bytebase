import { Code, ConnectError } from "@connectrpc/connect";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerIsReady: vi.fn(async () => {}),
  routerPush: vi.fn(),
  fetchServerInfo: vi.fn(async () => {}),
  listRoles: vi.fn(async () => {}),
  fetchIamPolicy: vi.fn(async () => {}),
  getOrFetchProjectByName: vi.fn(async (): Promise<{ name: string }> => {
    throw new ConnectError("not found", Code.NotFound);
  }),
  createProject: vi.fn(),
  setupSample: vi.fn(),
  updateWorkspaceProfile: vi.fn(),
  resetQuickstartProgress: vi.fn(),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    isReady: mocks.routerIsReady,
    push: mocks.routerPush,
  },
}));

vi.mock("@/stores/app", () => {
  const state = {
    enableOnboarding: () => true,
    appFeatures: { "bb.feature.database-change-mode": 0 },
    fetchServerInfo: mocks.fetchServerInfo,
    loadWorkspaceProfile: vi.fn(async () => undefined),
    listRoles: mocks.listRoles,
    fetchWorkspaceIamPolicy: mocks.fetchIamPolicy,
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    createProject: mocks.createProject,
    setupSample: mocks.setupSample,
    updateWorkspaceProfile: mocks.updateWorkspaceProfile,
    resetQuickstartProgress: mocks.resetQuickstartProgress,
  };
  return {
    useAppStore: Object.assign(
      (selector?: (s: typeof state) => unknown) =>
        selector ? selector(state) : state,
      { getState: () => state }
    ),
  };
});

vi.mock("@/components/auth/AuthFooter", () => ({
  AuthFooter: () => null,
}));

vi.mock("@/components/ComponentPermissionGuard", () => ({
  ComponentPermissionGuard: ({ children }: { children: React.ReactNode }) =>
    children,
}));

vi.mock("@/components/ResourceIdField", async () => {
  const React = await import("react");
  return {
    ResourceIdField: ({
      validate,
      onValidationChange,
    }: {
      validate?: (
        value: string
      ) => Promise<{ type: string; message: string }[]>;
      onValidationChange?: (valid: boolean) => void;
    }) => {
      React.useEffect(() => {
        validate?.("new-project").then((messages) => {
          onValidationChange?.(messages.length === 0);
        });
      }, [validate, onValidationChange]);
      return React.createElement("div", {
        "data-testid": "resource-id-field",
      });
    },
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

vi.mock("@/utils/connect", () => ({
  getErrorCode: (error: unknown) =>
    error instanceof ConnectError ? error.code : Code.Unknown,
  extractGrpcErrorMessage: (error: unknown) =>
    error instanceof ConnectError ? error.message : String(error),
}));

let SetupPage: typeof import("./SetupPage").SetupPage;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flushPromises = () =>
  act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });

beforeEach(async () => {
  vi.clearAllMocks();
  ({ SetupPage } = await import("./SetupPage"));
});

describe("SetupPage", () => {
  test("finishes setup by navigating to the workspace landing page", async () => {
    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const buttonByText = (text: string) =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === text
      );

    act(() => {
      buttonByText("→")?.click();
    });
    await flushPromises();

    act(() => {
      buttonByText("→")?.click();
    });
    await flushPromises();

    act(() => {
      buttonByText("common.confirm")?.click();
    });
    await flushPromises();

    expect(mocks.routerPush).toHaveBeenLastCalledWith({
      name: "workspace.landing",
    });
    expect(mocks.resetQuickstartProgress).toHaveBeenCalledTimes(1);

    unmount();
  });

  test("aligns the built-in sample radio with its title and styles the description", async () => {
    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const nextButton = () =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === "→"
      );

    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    const title = Array.from(container.querySelectorAll("div")).find(
      (element) => element.textContent === "setup.data.built-in"
    );
    const description = Array.from(container.querySelectorAll("div")).find(
      (element) => element.textContent === "setup.data.built-in-desc"
    );
    const option = title?.closest("label");
    const radio = option?.querySelector("[role='radio']");

    expect(option?.className).toContain("items-start");
    expect(radio?.className).toContain("mt-0.5");
    expect(title?.className).toContain("font-medium");
    expect(description?.className).toContain("text-control");

    unmount();
  });

  test("aligns default landing page radios with their titles and styles descriptions", async () => {
    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const nextButton = () =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === "→"
      );

    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    const workspaceTitle = Array.from(container.querySelectorAll("div")).find(
      (element) =>
        element.textContent ===
        "settings.general.workspace.default-landing-page.workspace.self"
    );
    const workspaceDescription = Array.from(
      container.querySelectorAll("div")
    ).find(
      (element) =>
        element.textContent ===
        "settings.general.workspace.default-landing-page.workspace.description"
    );
    const option = workspaceTitle?.closest("label");
    const radio = option?.querySelector("[role='radio']");

    expect(option?.className).toContain("items-start");
    expect(radio?.className).toContain("mt-0.5");
    expect(workspaceTitle?.className).toContain("font-medium");
    expect(workspaceDescription?.className).toContain("text-control");

    unmount();
  });

  test("allows advancing with a new project resource id", async () => {
    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const nextButton = () =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === "→"
      );

    expect(nextButton()).toBeTruthy();
    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith(
      "projects/new-project",
      true
    );
    expect(nextButton()?.disabled).toBe(false);

    unmount();
  });

  test("allows advancing when project lookup resolves to the unknown placeholder", async () => {
    mocks.getOrFetchProjectByName.mockResolvedValueOnce({
      name: "projects/-1",
    });

    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const nextButton = () =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === "→"
      );

    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith(
      "projects/new-project",
      true
    );
    expect(nextButton()?.disabled).toBe(false);

    unmount();
  });

  test("blocks advancing when project lookup fails for reasons other than NotFound", async () => {
    mocks.getOrFetchProjectByName.mockRejectedValueOnce(
      new ConnectError("backend unavailable", Code.Unavailable)
    );

    const { container, render, unmount } = renderIntoContainer(<SetupPage />);
    render();
    await flushPromises();

    const nextButton = () =>
      Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(
        (button) => button.textContent === "→"
      );

    expect(nextButton()).toBeTruthy();
    act(() => {
      nextButton()?.click();
    });
    await flushPromises();

    expect(nextButton()).toBeTruthy();
    expect(nextButton()?.disabled).toBe(true);

    unmount();
  });
});
