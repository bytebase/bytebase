import { redirect } from "react-router-dom";
import { useAppStore } from "@/react/stores/app";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  AUTH_2FA_SETUP_MODULE,
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_PROFILE_SETUP_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  OAUTH2_CONSENT_MODULE,
  PROJECT_V1_ROUTE_DASHBOARD,
  SETTING_ROUTE,
  SETUP_MODULE,
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "./handles";
import { resolvePath } from "./navigation";

const SIGNIN_QUERY_PARAMS = [
  "idp",
  "workspace",
  "email",
  "token",
  "invitation",
] as const;

// Auth/landing route names that don't require an authenticated session.
// Inlined (rather than importing `@/utils/auth`, which pulls the Vue router).
export function isAuthRelatedRoute(routeName: string): boolean {
  return [
    AUTH_SIGNIN_MODULE,
    AUTH_SIGNIN_ADMIN_MODULE,
    AUTH_SIGNUP_MODULE,
    AUTH_MFA_MODULE,
    AUTH_PASSWORD_RESET_MODULE,
    AUTH_PASSWORD_FORGOT_MODULE,
    AUTH_OAUTH_CALLBACK_MODULE,
    AUTH_OIDC_CALLBACK_MODULE,
  ].includes(routeName);
}

function stripSigninQueryParams(fullPath: string): string {
  const url = new URL(fullPath, window.location.origin);
  for (const param of SIGNIN_QUERY_PARAMS) {
    url.searchParams.delete(param);
  }
  return url.pathname + url.search + url.hash;
}

// Route-name prefixes that an authenticated user may always access.
const ALLOWED_ROUTE_PATTERNS = [
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  DATABASE_ROUTE_DASHBOARD,
  SETTING_ROUTE,
  SETUP_MODULE,
  "workspace",
  "sql-editor",
];

/**
 * Faithful port of the legacy vue-router `beforeEach` guard
 * (`src/router/index.ts`). Runs as the react-router root-route loader: the
 * root `.tsx` loader resolves the matched leaf route's `handle.name` (via
 * `matchRoutes`) and calls this. Returns a `redirect()` Response to navigate
 * elsewhere, or `null` to allow the navigation. Session state is read from the
 * app store (the single source of truth).
 */
export function rootGuard({
  name,
  url,
}: {
  name: string | undefined;
  url: URL;
}): Response | null {
  const toName = name ?? "";

  // Error pages can be accessed directly.
  if (toName === WORKSPACE_ROUTE_403 || toName === WORKSPACE_ROUTE_404) {
    return null;
  }
  // Auth callbacks can be accessed directly.
  if (
    toName === AUTH_OAUTH_CALLBACK_MODULE ||
    toName === AUTH_OIDC_CALLBACK_MODULE
  ) {
    return null;
  }
  // OAuth2 consent requires login but must not redirect away.
  if (toName === OAUTH2_CONSENT_MODULE) {
    return null;
  }

  const store = useAppStore.getState();
  const isLoggedIn = store.isLoggedIn();

  // Allow 2FA setup / password reset / profile setup for logged-in users.
  if (
    isLoggedIn &&
    (toName === AUTH_2FA_SETUP_MODULE ||
      toName === AUTH_PASSWORD_RESET_MODULE ||
      toName === AUTH_PROFILE_SETUP_MODULE)
  ) {
    return null;
  }

  // If logged in and trying to access auth pages, redirect to the main app.
  if (
    isAuthRelatedRoute(toName) &&
    isLoggedIn &&
    !store.unauthenticatedOccurred
  ) {
    const relayState = url.searchParams.get("relay_state") ?? undefined;
    const redirectParam = url.searchParams.get("redirect") ?? undefined;
    let target = "/";
    // Validate relay_state to prevent open redirect: relative URLs only,
    // reject protocol-relative (//).
    if (
      relayState &&
      relayState.startsWith("/") &&
      !relayState.startsWith("//")
    ) {
      target = relayState;
    } else if (redirectParam) {
      target = redirectParam;
    }
    return redirect(target);
  }

  // Auth pages: reset caches and allow access.
  if (isAuthRelatedRoute(toName)) {
    store.resetDatabases();
    store.resetInstances();
    store.resetProjects();
    void import("@/plugins/ai/store").then(({ useConversationStore }) => {
      useConversationStore.getState().reset();
    });
    return null;
  }

  // Require authentication for all other pages.
  if (!isLoggedIn) {
    const query: Record<string, string> = {};
    // Forward signin-only query params (consumed by the signin page).
    for (const param of SIGNIN_QUERY_PARAMS) {
      const value = url.searchParams.get(param);
      if (value) query[param] = value;
    }
    const fullPath = url.pathname + url.search + url.hash;
    // Set redirect if not root and not already set; strip signin-only params.
    if (fullPath !== "/" && !url.searchParams.get("redirect")) {
      query["redirect"] = stripSigninQueryParams(fullPath);
    }
    return redirect(resolvePath(AUTH_SIGNIN_MODULE, { query }));
  }

  // Enforce 2FA setup if required.
  const profile = store.getWorkspaceProfile();
  if (
    store.hasFeature(PlanFeature.FEATURE_TWO_FA) &&
    profile.requireMfa &&
    store.currentUser &&
    !store.currentUser.mfaEnabled &&
    toName !== AUTH_2FA_SETUP_MODULE
  ) {
    return redirect(resolvePath(AUTH_2FA_SETUP_MODULE));
  }

  // Enforce password reset if required.
  if (store.requireResetPassword() && toName !== AUTH_PASSWORD_RESET_MODULE) {
    return redirect(resolvePath(AUTH_PASSWORD_RESET_MODULE));
  }

  // Allow access to main application routes.
  if (ALLOWED_ROUTE_PATTERNS.some((pattern) => toName.startsWith(pattern))) {
    return null;
  }

  // An unnamed matched route (layout / redirect shell) is allowed — react
  // router only ran this loader because a route matched the URL.
  if (!toName) {
    return null;
  }

  // Fallback: unknown named route → 404.
  return redirect(resolvePath(WORKSPACE_ROUTE_404));
}
