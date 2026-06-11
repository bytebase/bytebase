import { Loader2 } from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useMatches, useNavigate } from "react-router-dom";
import { InactiveRemindModal } from "@/react/components/auth/InactiveRemindModal";
import {
  buildSigninRedirectQuery,
  isAuthRelatedRoute,
} from "@/react/router/guard";
import {
  AUTH_SIGNIN_MODULE,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "@/react/router/handles";
import { resolvePath } from "@/react/router/navigation";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import { isDev } from "@/utils";

// Session-validity poll interval (1 min dev, 5 min prod), mirroring the legacy
// AuthContext.vue.
const CHECK_AUTHORIZATION_INTERVAL = isDev() ? 60 * 1000 : 60 * 1000 * 5;

// Replaces AuthContext.vue: gates the app render on the authenticated session
// (loading workspace-scoped data first), polls session validity, redirects on a
// cross-tab user switch, and mounts the inactivity reminder. Reads session
// state from the app store (the single source of truth).
export function AuthGate({ children }: { children: ReactNode }) {
  const isLoggedIn = useAppStore((s) => s.isLoggedIn());
  const currentUser = useAppStore((s) => s.currentUser);
  const currentUserName = useAppStore((s) => s.currentUserName);
  const unauthenticatedOccurred = useAppStore((s) => s.unauthenticatedOccurred);

  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const matches = useMatches();
  const currentRouteName = (
    matches.at(-1)?.handle as { name?: string } | undefined
  )?.name;
  const isAuthRoute = Boolean(
    currentRouteName && isAuthRelatedRoute(currentRouteName)
  );
  const isPublicRoute =
    currentRouteName === WORKSPACE_ROUTE_403 ||
    currentRouteName === WORKSPACE_ROUTE_404;
  const currentUserWorkspace = currentUser?.workspace ?? "";
  const currentUserGroupsKey = (currentUser?.groups ?? []).join("\0");
  const currentPath = `${location.pathname}${location.search}${location.hash}`;
  const shouldRedirectToSignin = !isLoggedIn && !isAuthRoute && !isPublicRoute;

  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!shouldRedirectToSignin) return;
    navigate(
      resolvePath(AUTH_SIGNIN_MODULE, {
        query: buildSigninRedirectQuery(
          new URL(currentPath, window.location.origin)
        ),
      }),
      { replace: true }
    );
  }, [currentPath, navigate, shouldRedirectToSignin]);

  // Load workspace-scoped data once authenticated, then reveal the app.
  useEffect(() => {
    const store = useAppStore.getState();
    if (!isLoggedIn || !currentUserName) {
      setReady(true);
      return;
    }
    let cancelled = false;
    setReady(false);
    void Promise.all([
      store.loadSubscription(),
      store.fetchWorkspaceIamPolicy(),
      store.loadWorkspaceList(),
      store.listRoles(),
      store.batchGetOrFetchGroups(
        currentUserGroupsKey ? currentUserGroupsKey.split("\0") : []
      ),
    ]).finally(() => {
      if (!cancelled) setReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, [currentUserGroupsKey, currentUserName, currentUserWorkspace, isLoggedIn]);

  // Periodically revalidate the session (skip when logged out / on auth routes).
  useEffect(() => {
    const id = setInterval(() => {
      const store = useAppStore.getState();
      if (!store.isLoggedIn() || store.unauthenticatedOccurred) return;
      if (isAuthRoute) return;
      void (async () => {
        const user = await store.fetchCurrentUser();
        if (!user || !store.isLoggedIn() || store.unauthenticatedOccurred) {
          return;
        }
        await Promise.allSettled([
          store.fetchWorkspaceIamPolicy(),
          store.listRoles(),
        ]);
      })();
    }, CHECK_AUTHORIZATION_INTERVAL);
    return () => clearInterval(id);
  }, [isAuthRoute]);

  // Cross-tab user switch: when the signed-in user changes, redirect to the
  // workspace root with a notification (unless it's a self email update).
  const prevUserName = useRef(currentUserName);
  useEffect(() => {
    const prev = prevUserName.current;
    prevUserName.current = currentUserName;
    if (!currentUserName || !prev || currentUserName === prev) return;
    const store = useAppStore.getState();
    if (store.isSelfEmailUpdate) {
      store.setIsSelfEmailUpdate(false);
      return;
    }
    navigate(
      resolvePath(WORKSPACE_ROOT_MODULE, { query: { _r: `${Date.now()}` } })
    );
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("auth.login-as-another.title"),
      description: t("auth.login-as-another.content"),
    });
  }, [currentUserName, navigate, t]);

  // Auth surfaces (signin, OAuth/OIDC callbacks, MFA, …) render outside the
  // readiness gate: they don't depend on the workspace data loaded above, and
  // hiding them while it loads would unmount the OAuth callback page mid-login
  // (its `login()` flips `isLoggedIn`, which triggers that load) and blank the
  // signin page on first paint.
  return (
    <>
      {(ready || isAuthRoute) && !shouldRedirectToSignin ? (
        children
      ) : (
        <div className="flex items-center justify-center h-screen">
          <Loader2
            className="size-5 text-accent animate-spin"
            aria-label="Loading"
            role="status"
          />
        </div>
      )}
      {!isAuthRoute && isLoggedIn && !unauthenticatedOccurred && (
        <InactiveRemindModal />
      )}
    </>
  );
}
