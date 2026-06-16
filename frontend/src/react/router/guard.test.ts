import { matchRoutes } from "react-router-dom";
import { beforeEach, describe, expect, test, vi } from "vitest";

// Configurable fake session, controlled per test.
const session = {
  isLoggedIn: false,
  unauthenticatedOccurred: false,
  requireResetPassword: false,
  requireMfa: false,
  hasTwoFa: false,
  currentUser: undefined as { mfaEnabled: boolean } | undefined,
};

const resets = {
  resetDatabases: vi.fn(),
  resetInstances: vi.fn(),
  resetProjects: vi.fn(),
};

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      isLoggedIn: () => session.isLoggedIn,
      unauthenticatedOccurred: session.unauthenticatedOccurred,
      requireResetPassword: () => session.requireResetPassword,
      getWorkspaceProfile: () => ({ requireMfa: session.requireMfa }),
      hasFeature: () => session.hasTwoFa,
      currentUser: session.currentUser,
      ...resets,
    }),
  },
}));

vi.mock("@/plugins/ai/store", () => ({
  // Zustand store: the guard calls `useConversationStore.getState().reset()`.
  useConversationStore: { getState: () => ({ reset: vi.fn() }) },
}));

import { buildSigninRedirectQuery, rootGuard } from "./guard";
import {
  AUTH_2FA_SETUP_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_MODULE,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_404,
} from "./handles";
import { setRouteNameIndex } from "./navigation";
import { routes } from "./routes";

beforeEach(() => {
  session.isLoggedIn = false;
  session.unauthenticatedOccurred = false;
  session.requireResetPassword = false;
  session.requireMfa = false;
  session.hasTwoFa = false;
  session.currentUser = undefined;
  vi.clearAllMocks();
  setRouteNameIndex(
    new Map<string, string>([
      [AUTH_SIGNIN_MODULE, "/auth"],
      [AUTH_2FA_SETUP_MODULE, "/auth/2fa-setup"],
      [AUTH_PASSWORD_RESET_MODULE, "/auth/password-reset"],
      [WORKSPACE_ROUTE_404, "/404"],
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
    context: {},
  }) as Response | Promise<Response>;
}

describe("rootGuard", () => {
  test("error page is allowed directly", () => {
    expect(run(WORKSPACE_ROUTE_404, "/404")).toBeNull();
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
