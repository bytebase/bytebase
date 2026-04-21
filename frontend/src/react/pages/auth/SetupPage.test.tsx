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
  fetchRoleList: vi.fn(async () => {}),
  fetchIamPolicy: vi.fn(async () => {}),
  getOrFetchProjectByName: vi.fn(async () => {
    throw new ConnectError("not found", Code.NotFound);
  }),
  createProject: vi.fn(),
  setupSample: vi.fn(),
  updateWorkspaceProfile: vi.fn(),
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
}));

vi.mock("@/router", () => ({
  router: {
    isReady: mocks.routerIsReady,
    push: mocks.routerPush,
  },
}));

vi.mock("@/router/sqlEditor", () => ({
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
}));

vi.mock("@/store", () => ({
  useActuatorV1Store: () => ({
    enableOnboarding: true,
    setupSample: mocks.setupSample,
  }),
  useAppFeature: () => ({ value: 0 }),
  useProjectV1Store: () => ({
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    createProject: mocks.createProject,
  }),
  useRoleStore: () => ({
    fetchRoleList: mocks.fetchRoleList,
  }),
  useSettingV1Store: () => ({
    updateWorkspaceProfile: mocks.updateWorkspaceProfile,
  }),
  useWorkspaceV1Store: () => ({
    fetchIamPolicy: mocks.fetchIamPolicy,
  }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/components/auth/AuthFooter", () => ({
  AuthFooter: () => null,
}));

vi.mock("@/react/components/ComponentPermissionGuard", () => ({
  ComponentPermissionGuard: ({ children }: { children: React.ReactNode }) =>
    children,
}));

vi.mock("@/react/components/ResourceIdField", async () => {
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
