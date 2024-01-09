import { useTitle } from "@vueuse/core";
import { nextTick, ref } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import {
  hasFeature,
  useVCSV1Store,
  useSQLReviewStore,
  useSheetV1Store,
  useAuthStore,
  useActuatorV1Store,
  useRouterStore,
  useDBSchemaV1Store,
  useOnboardingStateStore,
  useTabStore,
  useIdentityProviderStore,
  useUserStore,
  useProjectV1Store,
  useProjectWebhookV1Store,
  useEnvironmentV1Store,
  useCurrentUserV1,
  useInstanceV1Store,
  useDatabaseV1Store,
  useChangeHistoryStore,
  useChangelistStore,
  usePageMode,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import {
  hasSettingPagePermission,
  hasWorkspacePermissionV1,
  idFromSlug,
  sheetNameFromSlug,
  uidFromSlug,
} from "@/utils";
import authRoutes, {
  AUTH_2FA_SETUP_MODULE,
  AUTH_MFA_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "./auth";
import dashboardRoutes from "./dashboard";
import { WORKSPACE_HOME_MODULE } from "./dashboard/workspace";
import sqlEditorRoutes, { SQL_EDITOR_HOME_MODULE } from "./sqlEditor";

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
  const dbSchemaStore = useDBSchemaV1Store();
  const routerStore = useRouterStore();
  const projectV1Store = useProjectV1Store();
  const projectWebhookV1Store = useProjectWebhookV1Store();

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
  if (to.name === "oauth-callback" || to.name === "oidc-callback") {
    next();
    return;
  }
  if (to.name === "workspace.debug-lsp") {
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
        path: `/sql-editor/sheet/project-sample-101`,
        replace: true,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.")) {
    const hasPermission = hasSettingPagePermission(
      to.name.toString(),
      currentUserV1.value.userRole
    );
    if (!hasPermission) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name === "workspace.instance") {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-instance",
        currentUserV1.value.userRole
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (
    to.name === "error.403" ||
    to.name === "error.404" ||
    to.name === "error.500" ||
    to.name === WORKSPACE_HOME_MODULE ||
    to.name === "workspace.inbox" ||
    to.name === "workspace.slow-query" ||
    to.name === "workspace.sync-schema" ||
    to.name === "workspace.export-center" ||
    to.name === "workspace.anomaly-center" ||
    to.name === "workspace.project" ||
    to.name === "workspace.instance" ||
    to.name === "workspace.database" ||
    to.name === "workspace.issue" ||
    to.name === "workspace.environment" ||
    to.name === SQL_EDITOR_HOME_MODULE ||
    (to.name?.toString().startsWith("setting") &&
      to.name?.toString() != "setting.workspace.gitops.detail" &&
      to.name?.toString() != "setting.workspace.sql-review.detail" &&
      to.name?.toString() != "setting.workspace.sso.detail")
  ) {
    next();
    return;
  }

  if (
    to.name === "workspace.project.detailV1" ||
    to.name === "workspace.project.database.dashboard" ||
    to.name === "workspace.project.issue.dashboard"
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

  if (to.name === "workspace.database.history.detail") {
    const parent = `instances/${to.params.instance}/databases/${to.params.database}`;
    const uid = uidFromSlug(to.params.changeHistorySlug as string);
    Promise.all([
      useDatabaseV1Store().getOrFetchDatabaseByName(parent),
      useChangeHistoryStore().fetchChangeHistory({
        name: `${parent}/changeHistories/${uid}`,
      }),
    ])
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

  const routerSlug = routerStore.routeSlug(to);
  const principalEmail = routerSlug.principalEmail;
  const environmentSlug = routerSlug.environmentSlug;
  const projectSlug = routerSlug.projectSlug;
  const projectWebhookSlug = routerSlug.projectWebhookSlug;
  const issueSlug = routerSlug.issueSlug;
  const instanceSlug = routerSlug.instanceSlug;
  const databaseSlug = routerSlug.databaseSlug;
  const vcsSlug = routerSlug.vcsSlug;
  const connectionSlug = routerSlug.connectionSlug;
  const sheetSlug = routerSlug.sheetSlug;
  const ssoName = routerSlug.ssoName;
  const sqlReviewPolicySlug = routerSlug.sqlReviewPolicySlug;

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

  if (environmentSlug) {
    useEnvironmentV1Store()
      .getOrFetchEnvironmentByUID(String(idFromSlug(environmentSlug)))
      .then((env) => {
        // getEnvironmentById returns unknown("ENVIRONMENT") when it doesn't exist
        // so we need to check the id here
        if (env && env.uid !== String(UNKNOWN_ID)) {
          next();
          return;
        } else {
          throw new Error();
        }
      })
      .catch(() => {
        next({
          name: "error.404",
          replace: false,
        });
      });
    return;
  }

  if (projectSlug && projectSlug !== "-") {
    projectV1Store
      .fetchProjectByUID(String(idFromSlug(projectSlug)))
      .then((project) => {
        if (projectWebhookSlug) {
          const webhook =
            projectWebhookV1Store.getProjectWebhookFromProjectById(
              project,
              idFromSlug(projectWebhookSlug)
            );
          if (webhook) {
            next();
          } else {
            next({
              name: "error.404",
              replace: false,
            });
            throw new Error("not found");
          }
        } else if (to.name === "workspace.project.changelist.detail") {
          const name = `${project.name}/changelists/${to.params.changelistName}`;
          useChangelistStore()
            .fetchChangelistByName(name)
            .then((changelist) => {
              if (changelist) {
                next();
              } else {
                next({
                  name: "error.404",
                  replace: false,
                });
                return;
              }
            })
            .catch((error) => {
              next({
                name: "error.404",
                replace: false,
              });
              throw error;
            });
        } else if (
          to.name === "workspace.project.branch.detail" &&
          to.params.branchName !== "new"
        ) {
          next();
          // const name = `${project.name}/branches/${to.params.branchName}`;
          // useBranchStore()
          //   .fetchBranchByName(name, false /* !useCache */)
          //   .then((branch) => {
          //     if (branch) {
          //       next();
          //     } else {
          //       next({
          //         name: "error.404",
          //         replace: false,
          //       });
          //       throw new Error("not found");
          //     }
          //   })
          //   .catch((error) => {
          //     next({
          //       name: "error.404",
          //       replace: false,
          //     });
          //     throw error;
          //   });
        } else {
          next();
        }
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

  if (to.name === "workspace.project.branch.detail") {
    if (to.params.branchName === "new") {
      next();
      return;
    }
  }

  if (databaseSlug) {
    if (databaseSlug.toLowerCase() == "grant") {
      next();
      return;
    }
    useDatabaseV1Store()
      .fetchDatabaseByUID(String(idFromSlug(databaseSlug)))
      .then((database) => {
        dbSchemaStore
          .getOrFetchDatabaseMetadata({
            database: database.name,
            skipCache: false,
            view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
          })
          .then(() => {
            next();
          });
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

  if (instanceSlug) {
    useInstanceV1Store()
      .getOrFetchInstanceByUID(String(idFromSlug(instanceSlug)))
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

  if (vcsSlug) {
    useVCSV1Store()
      .fetchVCSByUid(idFromSlug(vcsSlug))
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

  if (sqlReviewPolicySlug) {
    useSQLReviewStore()
      .getOrFetchReviewPolicyByEnvironmentUID(
        String(idFromSlug(sqlReviewPolicySlug))
      )
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

  if (sheetSlug) {
    const sheetName = sheetNameFromSlug(sheetSlug);
    useSheetV1Store()
      .fetchSheetByName(sheetName)
      .then(() => next())
      .catch(() => next());
    return;
  }

  if (ssoName) {
    useIdentityProviderStore()
      .getOrFetchIdentityProviderByName(unescape(ssoName))
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
