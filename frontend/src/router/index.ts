import { useTitle } from "@vueuse/core";
import { nextTick, ref } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import {
  hasFeature,
  useSheetV1Store,
  useAuthStore,
  useActuatorV1Store,
  useRouterStore,
  useOnboardingStateStore,
  useTabStore,
  useUserStore,
  useCurrentUserV1,
  usePageMode,
} from "@/store";
import { sheetNameFromSlug } from "@/utils";
import authRoutes, {
  AUTH_2FA_SETUP_MODULE,
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "./auth";
import dashboardRoutes from "./dashboard";
import { INSTANCE_ROUTE_DETAIL } from "./dashboard/instance";
import { ISSUE_ROUTE_DASHBOARD } from "./dashboard/issue";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_HOME_MODULE,
  WORKSPACE_ROUTE_SLOW_QUERY,
  WORKSPACE_ROUTE_EXPORT_CENTER,
  WORKSPACE_ROUTE_ANOMALY_CENTER,
} from "./dashboard/workspaceRoutes";
import { SETTING_ROUTE } from "./dashboard/workspaceSetting";
import sqlEditorRoutes, {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_SHARE_MODULE,
} from "./sqlEditor";

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [...authRoutes, ...dashboardRoutes, ...sqlEditorRoutes],
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

  const pageMode = usePageMode();

  // In standalone mode, we don't want to user get out of some standalone pages.
  if (pageMode.value === "STANDALONE") {
    // If user is trying to navigate away from SQL Editor, we'll explicitly return false to cancel the navigation.
    if (
      from.name?.toString().startsWith("sql-editor") &&
      !to.name?.toString().startsWith("sql-editor")
    ) {
      return false;
    }
  }

  const authStore = useAuthStore();
  const routerStore = useRouterStore();
  const isLoggedIn = authStore.isLoggedIn();

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : WORKSPACE_HOME_MODULE;
  const toModule = to.name
    ? to.name.toString().split(".")[0]
    : WORKSPACE_HOME_MODULE;

  if (toModule != fromModule) {
    routerStore.setBackPath(from.fullPath);
  }

  // SSO callback routes are relayes to handle the IdP callback and dispatch the subsequent events.
  // They are called in the following scenarios:
  // - Login via OAuth / OIDC
  // - Setup VCS provider
  // - Setup GitOps workflow in a project
  if (
    to.name === AUTH_OAUTH_CALLBACK_MODULE ||
    to.name === AUTH_OIDC_CALLBACK_MODULE
  ) {
    next();
    return;
  }

  if (
    to.name === AUTH_SIGNIN_MODULE ||
    to.name === AUTH_SIGNUP_MODULE ||
    to.name === AUTH_MFA_MODULE ||
    to.name === AUTH_PASSWORD_FORGOT_MODULE
  ) {
    useTabStore().reset();
    import("@/plugins/ai/store").then(({ useConversationStore }) => {
      useConversationStore().reset();
    });
    if (isLoggedIn) {
      if (typeof to.query.redirect === "string") {
        location.replace(to.query.redirect);
        return;
      }
      next({ name: WORKSPACE_HOME_MODULE, replace: true });
    } else {
      next();
    }
    return;
  } else {
    if (!isLoggedIn) {
      const query: any = {};
      if (to.fullPath !== "/") {
        query["redirect"] = to.fullPath;
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
  const serverInfo = useActuatorV1Store().serverInfo;

  // If 2FA is required, redirect to MFA setup page if the user has not enabled 2FA.
  if (hasFeature("bb.feature.2fa") && serverInfo?.require2fa) {
    const user = currentUserV1.value;
    if (user && !user.mfaEnabled) {
      next({
        name: AUTH_2FA_SETUP_MODULE,
        replace: true,
      });
      return;
    }
  }

  if (to.name === SQL_EDITOR_HOME_MODULE) {
    const onboardingStateStore = useOnboardingStateStore();
    if (onboardingStateStore.getStateByKey("sql-editor")) {
      // Open the "Sample Sheet" when the first time onboarding SQL Editor
      onboardingStateStore.consume("sql-editor");
      next({
        name: SQL_EDITOR_SHARE_MODULE,
        params: {
          sheetSlug: "project-sample-101",
        },
        replace: true,
      });
      return;
    }
  }

  if (
    to.name === "error.403" ||
    to.name === "error.404" ||
    to.name === "error.500" ||
    to.name === WORKSPACE_HOME_MODULE ||
    to.name === WORKSPACE_ROUTE_SLOW_QUERY ||
    to.name === WORKSPACE_ROUTE_EXPORT_CENTER ||
    to.name === WORKSPACE_ROUTE_ANOMALY_CENTER ||
    to.name?.toString().startsWith(ENVIRONMENT_V1_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(INSTANCE_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(PROJECT_V1_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(DATABASE_ROUTE_DASHBOARD) ||
    to.name?.toString().startsWith(ISSUE_ROUTE_DASHBOARD) ||
    to.name === INSTANCE_ROUTE_DETAIL ||
    to.name === SQL_EDITOR_HOME_MODULE ||
    to.name?.toString().startsWith(SETTING_ROUTE)
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
  const principalEmail = routerSlug.principalEmail;
  const issueSlug = routerSlug.issueSlug;
  const connectionSlug = routerSlug.connectionSlug;
  const sheetSlug = routerSlug.sheetSlug;

  if (principalEmail) {
    useUserStore()
      .getOrFetchUserById(principalEmail)
      .then(() => {
        next();
      })
      .catch((error) => {
        next({
          name: "error.404",
          replace: false,
        });
        throw error;
      });
    return;
  }

  if (issueSlug) {
    // We've moved the preparation data fetch jobs into IssueDetail page
    // so just next() here.
    next();
    return;
  }

  if (sheetSlug) {
    const sheetName = sheetNameFromSlug(sheetSlug);
    useSheetV1Store()
      .fetchSheetByName(sheetName)
      .then(() => next())
      .catch(() => next());
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
    if (to.meta.overrideTitle) {
      return;
    }
    if (to.meta.title) {
      document.title = to.meta.title(to);
    } else {
      document.title = DEFAULT_DOCUMENT_TITLE;
    }
  });
});
