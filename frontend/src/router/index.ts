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
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "./auth";
import dashboardRoutes from "./dashboard";
import { INSTANCE_ROUTE_DETAIL } from "./dashboard/instance";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROOT_MODULE,
} from "./dashboard/workspaceRoutes";
import { SETTING_ROUTE } from "./dashboard/workspaceSetting";
import setupRoutes from "./setup";
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

  const actuatorStore = useActuatorV1Store();
  const authStore = useAuthStore();
  const routerStore = useRouterStore();

  if (
    isAuthRelatedRoute(to.name as string) &&
    authStore.isLoggedIn &&
    !authStore.unauthenticatedOccurred &&
    !authStore.requireResetPassword
  ) {
    const redirect = (to.query["redirect"] as string) || "/";
    next(redirect);
    return;
  }

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : WORKSPACE_ROOT_MODULE;
  const toModule = to.name
    ? to.name.toString().split(".")[0]
    : WORKSPACE_ROOT_MODULE;

  if (toModule != fromModule) {
    routerStore.setBackPath(from.fullPath);
  }

  if (
    to.name === "error.403" ||
    to.name === "error.404" ||
    to.name === "error.500"
  ) {
    next();
    return;
  }

  // SSO callback routes are relayes to handle the IdP callback and dispatch the subsequent events.
  // They are called in the following scenarios:
  // - Login via OAuth / OIDC
  if (
    to.name === AUTH_OAUTH_CALLBACK_MODULE ||
    to.name === AUTH_OIDC_CALLBACK_MODULE
  ) {
    next();
    return;
  }

  if (
    to.name === AUTH_SIGNIN_MODULE ||
    to.name === AUTH_SIGNIN_ADMIN_MODULE ||
    to.name === AUTH_SIGNUP_MODULE ||
    to.name === AUTH_MFA_MODULE ||
    to.name === AUTH_PASSWORD_RESET_MODULE ||
    to.name === AUTH_PASSWORD_FORGOT_MODULE
  ) {
    useDatabaseV1Store().reset();
    useProjectV1Store().reset();
    useInstanceV1Store().reset();
    import("@/plugins/ai/store").then(({ useConversationStore }) => {
      useConversationStore().reset();
    });
    next();
    return;
  } else {
    if (!authStore.isLoggedIn) {
      const query: LocationQueryRaw = {
        ...(to.query || {}),
      };
      if (to.fullPath !== "/") {
        if (to.query["idp"]) {
          // TODO: remove query param `idp` from fullPath.
          query["idp"] = to.query["idp"];
        }
        if (!to.query["redirect"]) {
          query["redirect"] = to.fullPath;
        }
      }

      next({
        name: AUTH_SIGNIN_MODULE,
        query: query,
        replace: true,
      });
      return;
    }
  }

  // If there is a `redirect` in query param and prev page is signin or signup, redirect to the target route
  if (
    (from.name === AUTH_SIGNIN_MODULE || from.name === AUTH_SIGNUP_MODULE) &&
    typeof from.query.redirect === "string"
  ) {
    window.location.href = from.query.redirect;
    return;
  }

  if (to.name === AUTH_2FA_SETUP_MODULE) {
    next();
    return;
  }

  const currentUserV1 = useCurrentUserV1();

  // If 2FA is required, redirect to MFA setup page if the user has not enabled 2FA.
  if (
    hasFeature(PlanFeature.FEATURE_TWO_FA) &&
    actuatorStore.serverInfo?.require2fa
  ) {
    const user = currentUserV1.value;
    if (user && !user.mfaEnabled) {
      next({
        name: AUTH_2FA_SETUP_MODULE,
        replace: true,
      });
      return;
    }
  }

  if (
    to.name?.toString().startsWith(ENVIRONMENT_V1_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(INSTANCE_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(PROJECT_V1_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(DATABASE_ROUTE_DASHBOARD) ||
    to.name === INSTANCE_ROUTE_DETAIL ||
    to.name?.toString().startsWith("sql-editor") ||
    to.name?.toString().startsWith(SETTING_ROUTE) ||
    to.name?.toString().startsWith("workspace") ||
    to.name?.toString().startsWith("setup") ||
    to.name === AUTH_PASSWORD_RESET_MODULE
  ) {
    next();
    return;
  }

  // We may just change the anchor (e.g. in Issue Detail view), thus we don't need
  // to fetch the data to verify its existence since we have already verified before.
  if (to.path == from.path) {
    next();
    return;
  }

  const routerSlug = routerStore.routeSlug(to);
  const issueSlug = routerSlug.issueSlug;
  const connectionSlug = routerSlug.connectionSlug;
  const sheetSlug = routerSlug.sheetSlug;

  if (issueSlug) {
    // We've moved the preparation data fetch jobs into IssueDetail page
    // so just next() here.
    next();
    return;
  }

  if (sheetSlug) {
    // We've moved the preparation data fetch jobs into ProvideSQLEditorContext.
    // so just next() here.
    next();
    return;
  }

  if (connectionSlug) {
    // We've moved the preparation data fetch jobs into ProvideSQLEditorContext.
    // so just next() here.
    next();
    return;
  }

  next({
    name: "error.404",
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
