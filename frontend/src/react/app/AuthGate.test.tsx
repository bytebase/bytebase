import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  loadSubscription: vi.fn(async () => undefined),
  fetchWorkspaceIamPolicy: vi.fn(async () => undefined),
  loadWorkspaceList: vi.fn(async () => undefined),
  listRoles: vi.fn(async () => undefined),
  batchGetOrFetchGroups: vi.fn(async () => undefined),
  fetchCurrentUser: vi.fn(),
  setIsSelfEmailUpdate: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("react-router-dom", () => ({
  useMatches: () => [{ handle: { name: "workspace.dashboard" } }],
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/react/components/auth/InactiveRemindModal", () => ({
  InactiveRemindModal: () => <div data-testid="inactive-remind-modal" />,
}));

vi.mock("@/react/router/guard", () => ({
  isAuthRelatedRoute: () => false,
}));

vi.mock("@/react/router/handles", () => ({
  WORKSPACE_ROOT_MODULE: "workspace.root",
}));

vi.mock("@/react/router/navigation", () => ({
  resolvePath: () => "/",
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  isDev: () => true,
}));

vi.mock("@/react/stores/app", async () => {
  const { create } = await import("zustand");
  const initialUser = {
    name: "users/alice@example.com",
    email: "alice@example.com",
    workspace: "workspaces/default",
    groups: ["groups/dev"],
  };
  mocks.fetchCurrentUser.mockImplementation(async () => initialUser);
  const useAppStore = create(() => ({
    currentUser: initialUser,
    currentUserName: initialUser.name,
    unauthenticatedOccurred: false,
    isSelfEmailUpdate: false,
    isLoggedIn: () => true,
    loadSubscription: mocks.loadSubscription,
    fetchWorkspaceIamPolicy: mocks.fetchWorkspaceIamPolicy,
    loadWorkspaceList: mocks.loadWorkspaceList,
    listRoles: mocks.listRoles,
    batchGetOrFetchGroups: mocks.batchGetOrFetchGroups,
    fetchCurrentUser: mocks.fetchCurrentUser,
    setIsSelfEmailUpdate: mocks.setIsSelfEmailUpdate,
  }));
  return { useAppStore };
});

import { useAppStore } from "@/react/stores/app";
import { AuthGate } from "./AuthGate";

describe("AuthGate", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    vi.useFakeTimers();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  test("keeps the page visible when currentUser is refreshed with the same identity", async () => {
    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });

    expect(container.querySelector("[data-testid='page']")).not.toBeNull();
    expect(container.querySelector("[role='status']")).toBeNull();

    act(() => {
      const currentUser = useAppStore.getState().currentUser;
      expect(currentUser).toBeDefined();
      useAppStore.setState({
        currentUser: {
          ...currentUser,
        } as NonNullable<typeof currentUser>,
        currentUserName: currentUser!.name,
      });
    });

    expect(container.querySelector("[data-testid='page']")).not.toBeNull();
    expect(container.querySelector("[role='status']")).toBeNull();
  });

  test("refreshes permission data in the background during session polling", async () => {
    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });
    vi.clearAllMocks();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_000);
    });

    expect(mocks.fetchCurrentUser).toHaveBeenCalledTimes(1);
    expect(mocks.fetchWorkspaceIamPolicy).toHaveBeenCalledTimes(1);
    expect(mocks.listRoles).toHaveBeenCalledTimes(1);
    expect(container.querySelector("[data-testid='page']")).not.toBeNull();
    expect(container.querySelector("[role='status']")).toBeNull();
  });
});
