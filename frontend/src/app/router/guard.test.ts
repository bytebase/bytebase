import { matchRoutes, RouterContextProvider } from "react-router";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";

// Configurable fake session, controlled per test.
const session = {
  isLoggedIn: false,
  unauthenticatedOccurred: false,
  requireResetPassword: false,
  requireMfa: false,
  hasTwoFa: false,
  isSaaSMode: false,
  disallowSignup: false,
  currentUser: undefined as { mfaEnabled: boolean } | undefined,
  // Mirrors the store default: PIPELINE until the workspace profile loads.
  databaseChangeMode: DatabaseChangeMode.PIPELINE,
};

const resets = {
  resetDatabases: vi.fn(),
  resetInstances: vi.fn(),
  resetProjects: vi.fn(),
};

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      isLoggedIn: () => session.isLoggedIn,
      unauthenticatedOccurred: session.unauthenticatedOccurred,
      requireResetPassword: () => session.requireResetPassword,
      getWorkspaceProfile: () => ({ requireMfa: session.requireMfa }),
      hasFeature: () => session.hasTwoFa,
      isSaaSMode: () => session.isSaaSMode,
      serverInfo: {
        restriction: { disallowSignup: session.disallowSignup },
      },
      currentUser: session.currentUser,
      appFeatures: {
        "bb.feature.database-change-mode": session.databaseChangeMode,
      },
      ...resets,
    }),
  },
}));

vi.mock("@/modules/ai/store", () => ({
  // Zustand store: the guard calls `useConversationStore.getState().reset()`.
  useConversationStore: { getState: () => ({ reset: vi.fn() }) },
}));

import { buildSigninRedirectQuery, rootGuard } from "./guard";
import {
  AUTH_2FA_SETUP_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
  PROJECT_V1_ROUTE_DASHBOARD,
  SQL_EDITOR_HOME_MODULE,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_404,
  WORKSPACE_ROUTE_LANDING,
} from "./handles";
import { setRouteNameIndex } from "./navigation";
import { routes } from "./routes";

beforeEach(() => {
  session.isLoggedIn = false;
  session.unauthenticatedOccurred = false;
  session.requireResetPassword = false;
  session.requireMfa = false;
  session.hasTwoFa = false;
  session.isSaaSMode = false;
  session.disallowSignup = false;
  session.currentUser = undefined;
  session.databaseChangeMode = DatabaseChangeMode.PIPELINE;
  vi.clearAllMocks();
  setRouteNameIndex(
    new Map<string, string>([
      [AUTH_SIGNIN_MODULE, "/auth"],
      [AUTH_2FA_SETUP_MODULE, "/auth/2fa-setup"],
      [AUTH_PASSWORD_RESET_MODULE, "/auth/password-reset"],
      [WORKSPACE_ROUTE_404, "/404"],
      [SQL_EDITOR_HOME_MODULE, "/sql-editor"],
      [WORKSPACE_ROUTE_LANDING, "/landing"],
    ])
  );
});

const loc = (location: (typeof window)["location"] | undefined) => location;
void loc; // keep TS happy if unused

function run(name: string | undefined, path: string) {
  return rootGuard({ name, url: new URL(`https://app.example.com${path}`) });
}

function location(result: Response | null): string | null {
  return result instanceof Response ? result.headers.get("Location") : null;
}

async function runCatchAllLoader(path: string): Promise<Response> {
  const matched = matchRoutes(routes, path);
  const leafRoute = matched?.at(-1)?.route;
  if (typeof leafRoute?.loader !== "function") {
    throw new Error(`No loader matched ${path}`);
  }
  const url = new URL(`https://app.example.com${path}`);
  return leafRoute.loader({
    request: new Request(url),
    url,
    pattern: "*",
    params: {},
    context: new RouterContextProvider(),
  }) as Response | Promise<Response>;
}

describe("rootGuard", () => {
  test("error page is allowed directly", () => {
    expect(run(WORKSPACE_ROUTE_404, "/404")).toBeNull();
  });

  test("root sends an EDITOR workspace to the SQL Editor", () => {
    session.isLoggedIn = true;
    session.databaseChangeMode = DatabaseChangeMode.EDITOR;

    expect(location(run(WORKSPACE_ROOT_MODULE, "/"))).toBe("/sql-editor");
  });

  // Documents why `login()` must load the workspace profile before it
  // navigates: this guard cannot tell "not loaded" from "PIPELINE", and the
  // last-visit fallback ignores /sql-editor, so an EDITOR workspace whose
  // profile is still unloaded lands here instead of the editor.
  test("root falls back to landing while the workspace profile is unloaded", () => {
    session.isLoggedIn = true;
    session.databaseChangeMode = DatabaseChangeMode.PIPELINE;

    expect(location(run(WORKSPACE_ROOT_MODULE, "/"))).toBe("/landing");
  });

  test("logged-out user on an unknown URL matched by the 404 catch-all is redirected to signin", () => {
    const target = location(run(WORKSPACE_ROUTE_404, "/ioewjfiwoejf"));
    expect(target).toBe("/auth?redirect=%2Fioewjfiwoejf");
  });

  test("logged-out catch-all route loader redirects to signin before 404", async () => {
    const response = await runCatchAllLoader("/ioewjfiwoejf");
    expect(response.headers.get("Location")).toBe(
      "/auth?redirect=%2Fioewjfiwoejf"
    );
  });

  test("logged-in catch-all route loader redirects to 404", async () => {
    session.isLoggedIn = true;
    const response = await runCatchAllLoader("/ioewjfiwoejf");
    expect(response.headers.get("Location")).toBe("/404");
  });

  test("/auth/admin matches the catch-all route", () => {
    const matched = matchRoutes(routes, "/auth/admin");
    const leafRoute = matched?.at(-1)?.route;
    const handle = leafRoute?.handle as { name?: string } | undefined;
    expect(handle?.name).toBe(WORKSPACE_ROUTE_404);
  });

  test("oauth callback is allowed directly", () => {
    expect(run(AUTH_OAUTH_CALLBACK_MODULE, "/auth/oauth/callback")).toBeNull();
  });

  test("logged-in user on 2FA-setup route is allowed", () => {
    session.isLoggedIn = true;
    expect(run(AUTH_2FA_SETUP_MODULE, "/auth/2fa-setup")).toBeNull();
  });

  test("logged-in user on the signin route is redirected home", () => {
    session.isLoggedIn = true;
    expect(location(run(AUTH_SIGNIN_MODULE, "/auth"))).toBe("/");
  });

  test("logged-in user on signin with ?redirect goes there", () => {
    session.isLoggedIn = true;
    expect(location(run(AUTH_SIGNIN_MODULE, "/auth?redirect=/projects"))).toBe(
      "/projects"
    );
  });

  test("auth route resets caches and allows access", () => {
    expect(run(AUTH_SIGNIN_MODULE, "/auth")).toBeNull();
    expect(resets.resetDatabases).toHaveBeenCalled();
    expect(resets.resetInstances).toHaveBeenCalled();
    expect(resets.resetProjects).toHaveBeenCalled();
  });

  test("redirects to signin when signup is disallowed", () => {
    session.disallowSignup = true;

    expect(
      location(
        run(
          AUTH_SIGNUP_MODULE,
          "/auth/signup?email=alice%40example.com&invitation=invite-1"
        )
      )
    ).toBe("/auth?email=alice%40example.com&invitation=invite-1");
  });

  test("not-logged-in user is redirected to signin with a redirect query", () => {
    const target = location(run(PROJECT_V1_ROUTE_DASHBOARD, "/projects/p1"));
    expect(target).toBe("/auth?redirect=%2Fprojects%2Fp1");
  });

  test("builds signin query while stripping signin-only params from redirect", () => {
    expect(
      buildSigninRedirectQuery(
        new URL(
          "https://app.example.com/projects?idp=idp-1&email=alice%40example.com&foo=bar&invitation=invite-1#section"
        )
      )
    ).toEqual({
      idp: "idp-1",
      email: "alice@example.com",
      invitation: "invite-1",
      redirect: "/projects?foo=bar#section",
    });
  });

  test("enforces 2FA setup when required", () => {
    session.isLoggedIn = true;
    session.hasTwoFa = true;
    session.requireMfa = true;
    session.currentUser = { mfaEnabled: false };
    expect(location(run(PROJECT_V1_ROUTE_DASHBOARD, "/projects/p1"))).toBe(
      "/auth/2fa-setup"
    );
  });

  test("enforces password reset when required", () => {
    session.isLoggedIn = true;
    session.requireResetPassword = true;
    expect(location(run(PROJECT_V1_ROUTE_DASHBOARD, "/projects/p1"))).toBe(
      "/auth/password-reset"
    );
  });

  test("allows an authenticated user on an allowed route", () => {
    session.isLoggedIn = true;
    expect(run(PROJECT_V1_ROUTE_DASHBOARD, "/projects/p1")).toBeNull();
  });

  test("unknown named route falls back to 404", () => {
    session.isLoggedIn = true;
    expect(location(run("some.unknown.route", "/whatever"))).toBe("/404");
  });

  test("unnamed matched route is allowed", () => {
    session.isLoggedIn = true;
    expect(run(undefined, "/projects/p1/some-shell")).toBeNull();
  });
});
