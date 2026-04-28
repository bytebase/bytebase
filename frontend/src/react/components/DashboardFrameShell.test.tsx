import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  loadEnvironmentList: vi.fn(),
  loadWorkspaceProfile: vi.fn(),
  useAppStore: vi.fn(),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("./BannersWrapper", () => ({
  BannersWrapper: () => <div data-testid="banners-wrapper" />,
}));

let DashboardFrameShell: typeof import("./DashboardFrameShell").DashboardFrameShell;

beforeEach(async () => {
  mocks.loadEnvironmentList.mockReset();
  mocks.loadWorkspaceProfile.mockReset();
  mocks.loadEnvironmentList.mockResolvedValue([]);
  mocks.loadWorkspaceProfile.mockResolvedValue(undefined);
  mocks.useAppStore.mockImplementation((selector) =>
    selector({
      loadEnvironmentList: mocks.loadEnvironmentList,
      loadWorkspaceProfile: mocks.loadWorkspaceProfile,
    })
  );
  ({ DashboardFrameShell } = await import("./DashboardFrameShell"));
});

describe("DashboardFrameShell", () => {
  test("reports stable banner and body targets after initialization", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onReady = vi.fn();

    act(() => {
      root.render(<DashboardFrameShell onReady={onReady} />);
    });

    expect(onReady).not.toHaveBeenCalled();
    expect(container.querySelector(".animate-spin")).not.toBeNull();

    await act(async () => {
      await Promise.resolve();
    });

    expect(onReady).toHaveBeenCalled();
    const targets = onReady.mock.lastCall?.[0];
    expect(targets.banner).toBeInstanceOf(HTMLDivElement);
    expect(targets.body).toBeInstanceOf(HTMLDivElement);
    expect(container.querySelector(".h-screen")).not.toBeNull();
    expect(
      container.querySelector('[data-testid="banners-wrapper"]')
    ).not.toBeNull();
    expect(container.querySelector(".animate-spin")).toBeNull();

    act(() => {
      root.unmount();
    });
    container.remove();
  });

  test("keeps body target hidden while bootstrap requests are pending", () => {
    let resolveEnvironmentList: (value: []) => void = () => {};
    mocks.loadEnvironmentList.mockReturnValue(
      new Promise((resolve) => {
        resolveEnvironmentList = resolve;
      })
    );
    mocks.loadWorkspaceProfile.mockResolvedValue(undefined);
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onReady = vi.fn();

    act(() => {
      root.render(<DashboardFrameShell onReady={onReady} />);
    });

    expect(onReady).not.toHaveBeenCalled();
    expect(container.querySelector(".animate-spin")).not.toBeNull();

    act(() => {
      resolveEnvironmentList([]);
      root.unmount();
    });
    container.remove();
  });
});
