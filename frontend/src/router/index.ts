import { useTitle } from "@vueuse/core";
import { nextTick, ref } from "vue";
import {
  createRouter,
  createWebHistory,
  type LocationQueryRaw,
} from "vue-router";
import {
  hasFeature,
  useAuthStore,
  useActuatorV1Store,
  useRouterStore,
  useCurrentUserV1,
  useProjectV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
} from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { isAuthRelatedRoute } from "@/utils/auth";
import authRoutes, {
  AUTH_2FA_SETUP_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_MODULE,
  OAUTH2_CONSENT_MODULE,
} from "./auth";
import dashboardRoutes from "./dashboard";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "./dashboard/workspaceRoutes";
import { SETTING_ROUTE } from "./dashboard/workspaceSetting";
import setupRoutes, { SETUP_MODULE } from "./setup";
import sqlEditorRoutes from "./sqlEditor";

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    ...authRoutes,
    ...setupRoutes,
    ...dashboardRoutes,
    ...sqlEditorRoutes,
  ],
  linkExactActiveClass: "bg-link-hover",
  scrollBehavior(to /*, from, savedPosition */) {
    if (to.hash) {
      try {
        const el = document.querySelector(to.hash);
        if (el) {
          return {
            el: to.hash,
            behavior: "smooth",
          };
        }
      } catch {
        // nothing todo
      }
    }
  },
});

router.beforeEach((to, from, next) => {
  console.debug("Router %s -> %s", from.name, to.name);

  // === SPECIAL CASES - Allow direct access ===

  // Prevent infinite loops only for identical routes without parameter changes
  if (from.name === to.name && from.fullPath === to.fullPath) {
    next(false);
    return;
  }

  // Error pages can be accessed directly
  if (to.name === WORKSPACE_ROUTE_403 || to.name === WORKSPACE_ROUTE_404) {
    next();
    return;
  }

  // Auth callbacks can be accessed directly
  if (
    to.name === AUTH_OAUTH_CALLBACK_MODULE ||
    to.name === AUTH_OIDC_CALLBACK_MODULE
  ) {
    next();
    return;
  }

  // OAuth2 consent page requires login but should not redirect away
  if (to.name === OAUTH2_CONSENT_MODULE) {
    next();
    return;
  }

  const actuatorStore = useActuatorV1Store();
  const authStore = useAuthStore();
  const routerStore = useRouterStore();

  // Allow access to 2FA setup and password reset for logged-in users
  if (
    authStore.isLoggedIn &&
    (to.name === AUTH_2FA_SETUP_MODULE ||
      to.name === AUTH_PASSWORD_RESET_MODULE)
  ) {
    next();
    return;
  }

  // === MODULE NAVIGATION TRACKING ===

  const fromModule =
    from.name?.toString().split(".")[0] || WORKSPACE_ROOT_MODULE;
  const toModule = to.name?.toString().split(".")[0] || WORKSPACE_ROOT_MODULE;

  if (toModule !== fromModule) {
    routerStore.setBackPath(from.fullPath);
  }

  // === AUTHENTICATION LOGIC ===

  // If user is logged in and tries to access auth pages, redirect to main app
  if (
    isAuthRelatedRoute(to.name as string) &&
    authStore.isLoggedIn &&
    !authStore.unauthenticatedOccurred
  ) {
    // For IdP-initiated SSO, use relay_state parameter
    // For other auth routes, use redirect parameter
    const relayState = to.query["relay_state"] as string | undefined;
    const redirectParam = to.query["redirect"] as string | undefined;

    // Validate relay_state to prevent open redirect attacks
    let redirect = "/";
    if (relayState && typeof relayState === "string") {
      // Only allow relative URLs, reject protocol-relative URLs (//)
      if (relayState.startsWith("/") && !relayState.startsWith("//")) {
        redirect = relayState;
      }
    } else if (redirectParam) {
      redirect = redirectParam;
    }

    next(redirect);
    return;
  }

  // Auth pages: Reset stores and allow access
  if (isAuthRelatedRoute(to.name as string)) {
    useDatabaseV1Store().reset();
    useProjectV1Store().reset();
    useInstanceV1Store().reset();
    import("@/plugins/ai/store").then(({ useConversationStore }) => {
      useConversationStore().reset();
    });
    next();
    return;
  }

  // Require authentication for all other pages
  if (!authStore.isLoggedIn) {
    const query: LocationQueryRaw = {};

    // Preserve important query parameters
    if (to.query) {
      // Preserve IDP parameter for OAuth flows
      if (to.query["idp"]) {
        query["idp"] = to.query["idp"];
      }
      // Preserve other auth-related parameters
      if (to.query["token"]) {
        query["token"] = to.query["token"];
      }
      if (to.query["invitation"]) {
        query["invitation"] = to.query["invitation"];
      }
    }

    // Set redirect URL if not root path and not already set
    if (to.fullPath !== "/" && !to.query["redirect"]) {
      query["redirect"] = to.fullPath;
    }

    next({
      name: AUTH_SIGNIN_MODULE,
      query,
      replace: true,
    });
    return;
  }

  // Enforce 2FA setup if required
  const currentUserV1 = useCurrentUserV1();
  if (
    hasFeature(PlanFeature.FEATURE_TWO_FA) &&
    actuatorStore.serverInfo?.require2fa &&
    currentUserV1.value &&
    !currentUserV1.value.mfaEnabled &&
    to.name !== AUTH_2FA_SETUP_MODULE // Prevent redirect loop
  ) {
    next({
      name: AUTH_2FA_SETUP_MODULE,
      replace: true,
    });
    return;
  }

  // Enforce password reset if required
  if (
    authStore.requireResetPassword &&
    to.name !== AUTH_PASSWORD_RESET_MODULE // Prevent redirect loop
  ) {
    next({
      name: AUTH_PASSWORD_RESET_MODULE,
      replace: true,
    });
    return;
  }

  // === ROUTE ACCESS CONTROL ===

  // Allow access to main application routes
  const allowedRoutePatterns = [
    ENVIRONMENT_V1_ROUTE_DASHBOARD,
    INSTANCE_ROUTE_DASHBOARD,
    PROJECT_V1_ROUTE_DASHBOARD,
    DATABASE_ROUTE_DASHBOARD,
    SETTING_ROUTE,
    SETUP_MODULE,
    "workspace",
    "sql-editor",
  ];

  if (
    allowedRoutePatterns.some((pattern) =>
      to.name?.toString().startsWith(pattern)
    )
  ) {
    next();
    return;
  }

  // Allow same-path navigation (e.g. anchor changes)
  if (to.path === from.path) {
    next();
    return;
  }

  // === FALLBACK ===

  // Log unexpected route for debugging
  console.warn("Router: Unexpected route", {
    to: to.name,
    path: to.path,
    fullPath: to.fullPath,
  });

  next({
    name: WORKSPACE_ROUTE_404,
    replace: false,
  });
});

const DEFAULT_DOCUMENT_TITLE = "Bytebase";
const title = ref(DEFAULT_DOCUMENT_TITLE);
useTitle(title);

router.afterEach((to /*, from */) => {
  // Needs to use nextTick otherwise title will still be the one from the previous route.
  nextTick(() => {
    if (to.meta.title) {
      document.title = to.meta.title(to);
    } else {
      document.title = DEFAULT_DOCUMENT_TITLE;
    }
  });
});
