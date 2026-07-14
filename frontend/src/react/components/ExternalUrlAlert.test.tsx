import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  hasWorkspacePermissionV2: vi.fn(),
  routerPush: vi.fn(),
  useServerState: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router/handles", () => ({
  SETTING_ROUTE_WORKSPACE_GENERAL: "setting.workspace.general",
}));

vi.mock("@/react/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: JSON.stringify(to) }),
  },
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useServerState: mocks.useServerState,
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

let ExternalUrlAlert: typeof import("./ExternalUrlAlert").ExternalUrlAlert;

describe("ExternalUrlAlert", () => {
  beforeEach(async () => {
    mocks.hasWorkspacePermissionV2.mockReset();
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);
    mocks.routerPush.mockReset();
    mocks.useServerState.mockReset();
    mocks.useServerState.mockReturnValue({ needConfigureExternalUrl: true });
    ({ ExternalUrlAlert } = await import("./ExternalUrlAlert"));
  });

  test("routes configure action to workspace general settings with external URL intro", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<ExternalUrlAlert />);
    });

    const link = Array.from(container.querySelectorAll("a")).find(
      (element) => element.textContent === "common.configure-now"
    ) as HTMLAnchorElement;

    expect(link).toBeTruthy();

    act(() => {
      link.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "setting.workspace.general",
      query: { intro: "external-url" },
    });

    act(() => {
      root.unmount();
    });
  });

  test("hides configure action without workspace profile permission", () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);

    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<ExternalUrlAlert />);
    });

    expect(container.querySelector("a")).toBeNull();

    act(() => {
      root.unmount();
    });
  });

  test("renders nothing when external URL does not need configuration", () => {
    mocks.useServerState.mockReturnValue({ needConfigureExternalUrl: false });

    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<ExternalUrlAlert />);
    });

    expect(container.textContent).toBe("");

    act(() => {
      root.unmount();
    });
  });
});
