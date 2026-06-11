import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  session: {
    isLoggedIn: true,
  },
  navigate: vi.fn(),
  location: {
    pathname: "/projects",
    search: "",
    hash: "",
  },
  routeName: "workspace.dashboard",
  resolvePath: vi.fn(
    (
      _name: string,
      options?: { query?: Record<string, string | undefined> }
    ) => {
      const query = new URLSearchParams();
      for (const [key, value] of Object.entries(options?.query ?? {})) {
        if (value) query.set(key, value);
      }
      const serialized = query.toString();
      return serialized ? `/auth?${serialized}` : "/auth";
    }
  ),
  buildSigninRedirectQuery: vi.fn((url: URL) => {
    const query: Record<string, string> = {};
    for (const param of ["idp", "workspace", "email", "token", "invitation"]) {
      const value = url.searchParams.get(param);
      if (value) query[param] = value;
    }
    const redirectURL = new URL(url.toString());
    for (const param of ["idp", "workspace", "email", "token", "invitation"]) {
      redirectURL.searchParams.delete(param);
    }
    const redirectPath =
      redirectURL.pathname + redirectURL.search + redirectURL.hash;
    if (redirectPath !== "/" && !url.searchParams.get("redirect")) {
      query.redirect = redirectPath;
    }
    return query;
  }),
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
  useLocation: () => mocks.location,
  useMatches: () => [{ handle: { name: mocks.routeName } }],
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/react/components/auth/InactiveRemindModal", () => ({
  InactiveRemindModal: () => <div data-testid="inactive-remind-modal" />,
}));

vi.mock("@/react/router/guard", () => ({
  buildSigninRedirectQuery: mocks.buildSigninRedirectQuery,
  isAuthRelatedRoute: (name: string) => name.startsWith("auth."),
}));

vi.mock("@/react/router/handles", () => ({
  AUTH_SIGNIN_MODULE: "auth.signin",
  WORKSPACE_ROUTE_403: "error.403",
  WORKSPACE_ROUTE_404: "error.404",
  WORKSPACE_ROOT_MODULE: "workspace.root",
}));

vi.mock("@/react/router/navigation", () => ({
  resolvePath: mocks.resolvePath,
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
    isLoggedIn: () => mocks.session.isLoggedIn,
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
    mocks.session.isLoggedIn = true;
    mocks.routeName = "workspace.dashboard";
    mocks.location.pathname = "/projects";
    mocks.location.search = "";
    mocks.location.hash = "";
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

  test("redirects protected routes to signin when the session is logged out", async () => {
    mocks.session.isLoggedIn = false;

    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });

    expect(mocks.navigate).toHaveBeenCalledWith("/auth?redirect=%2Fprojects", {
      replace: true,
    });
    expect(container.querySelector("[data-testid='page']")).toBeNull();
    expect(container.querySelector("[role='status']")).not.toBeNull();
  });

  test("preserves signin query params outside the redirect target", async () => {
    mocks.session.isLoggedIn = false;
    mocks.location.search =
      "?idp=idp-1&email=alice%40example.com&foo=bar&invitation=invite-1";
    mocks.location.hash = "#section";

    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });

    expect(mocks.resolvePath).toHaveBeenCalledWith("auth.signin", {
      query: {
        idp: "idp-1",
        email: "alice@example.com",
        invitation: "invite-1",
        redirect: "/projects?foo=bar#section",
      },
    });
    expect(mocks.navigate).toHaveBeenCalledWith(
      "/auth?idp=idp-1&email=alice%40example.com&invitation=invite-1&redirect=%2Fprojects%3Ffoo%3Dbar%23section",
      { replace: true }
    );
  });

  test("keeps public error routes visible when the session is logged out", async () => {
    mocks.session.isLoggedIn = false;
    mocks.routeName = "error.404";
    mocks.location.pathname = "/404";

    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });

    expect(mocks.navigate).not.toHaveBeenCalled();
    expect(container.querySelector("[data-testid='page']")).not.toBeNull();
  });

  test("keeps auth-route children mounted while post-login workspace data loads", async () => {
    mocks.session.isLoggedIn = false;
    mocks.routeName = "auth.oauth.callback";
    mocks.location.pathname = "/oauth/callback";
    // Hold the post-login data load open so the gate stays in its loading
    // phase for the rest of the test (once: it must not leak into later tests
    // — clearAllMocks doesn't restore implementations).
    mocks.loadSubscription.mockImplementationOnce(() => new Promise(() => {}));

    await act(async () => {
      root.render(
        <AuthGate>
          <div data-testid="page">Page</div>
        </AuthGate>
      );
    });

    expect(container.querySelector("[data-testid='page']")).not.toBeNull();
    expect(mocks.navigate).not.toHaveBeenCalled();

    // The OAuth callback page completes login() mid-page: the session flips
    // without a navigation. The callback page must stay mounted — unmounting
    // would discard its in-flight UI and re-process the single-use OAuth
    // state token on remount.
    await act(async () => {
      mocks.session.isLoggedIn = true;
      useAppStore.setState({});
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
