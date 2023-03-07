import { nextTick, ref } from "vue";
import {
  createRouter,
  createWebHistory,
  RouteLocationNormalized,
  RouteRecordRaw,
} from "vue-router";
import { useTitle } from "@vueuse/core";

import BodyLayout from "../layouts/BodyLayout.vue";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import DatabaseLayout from "../layouts/DatabaseLayout.vue";
import InstanceLayout from "../layouts/InstanceLayout.vue";
import SplashLayout from "../layouts/SplashLayout.vue";
import SQLEditorLayout from "../layouts/SQLEditorLayout.vue";
import SheetDashboardLayout from "../layouts/SheetDashboardLayout.vue";
import { t } from "../plugins/i18n";
import {
  Database,
  DEFAULT_PROJECT_ID,
  QuickActionType,
  UNKNOWN_ID,
} from "../types";
import {
  hasProjectPermission,
  hasWorkspacePermission,
  idFromSlug,
  migrationHistoryIdFromSlug,
} from "../utils";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import DashboardSidebar from "../views/DashboardSidebar.vue";
import Home from "../views/Home.vue";
import {
  hasFeature,
  useVCSStore,
  useProjectWebhookStore,
  useDataSourceStore,
  useSQLReviewStore,
  useProjectStore,
  useSheetStore,
  useAuthStore,
  useDatabaseStore,
  useEnvironmentStore,
  useInstanceStore,
  usePrincipalStore,
  useRouterStore,
  useDBSchemaStore,
  useConnectionTreeStore,
  useOnboardingStateStore,
  useTabStore,
  useIdentityProviderStore,
  useCurrentUser,
} from "@/store";

const HOME_MODULE = "workspace.home";
const AUTH_MODULE = "auth";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const ACTIVATE_MODULE = "auth.activate";
const PASSWORD_RESET_MODULE = "auth.password.reset";
const PASSWORD_FORGOT_MODULE = "auth.password.forgot";
const SQL_EDITOR_HOME_MODULE = "sql-editor.home";

const routes: Array<RouteRecordRaw> = [
  {
    path: "/auth",
    name: AUTH_MODULE,
    component: SplashLayout,
    children: [
      {
        path: "",
        name: SIGNIN_MODULE,
        meta: { title: () => t("common.sign-in") },
        component: Signin,
        alias: "signin",
        props: true,
      },
      {
        path: "signup",
        name: SIGNUP_MODULE,
        meta: { title: () => t("common.sign-up") },
        component: Signup,
        props: true,
      },
      // TODO(tianzhou): Disable activate page for now, requires implementing invite
      // {
      //   path: "activate",
      //   name: ACTIVATE_MODULE,
      //   meta: { title: () => "Activate" },
      //   component: Activate,
      //   props: true,
      // },
      // {
      //   path: "password-reset",
      //   name: PASSWORD_RESET_MODULE,
      //   meta: { title: () => "Reset Password" },
      //   component: PasswordReset,
      //   props: true,
      // },
      {
        path: "password-forgot",
        name: PASSWORD_FORGOT_MODULE,
        meta: { title: () => `${t("auth.password-forgot")}` },
        component: () => import("../views/auth/PasswordForgot.vue"),
        props: true,
      },
    ],
  },
  {
    path: "/oauth/callback",
    name: "oauth-callback",
    component: () => import("../views/OAuthCallback.vue"),
  },
  {
    path: "/oidc/callback",
    name: "oidc-callback",
    component: () => import("../views/OAuthCallback.vue"),
  },
  {
    path: "/",
    component: DashboardLayout,
    children: [
      {
        path: "",
        components: { body: BodyLayout },
        children: [
          {
            path: "",
            name: HOME_MODULE,
            meta: {
              quickActionListByRole: () => {
                const hasDBAWorkflowFeature = hasFeature(
                  "bb.feature.dba-workflow"
                );
                const ownerList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                    ];
                const dbaList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                    ];
                const developerList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync",
                      "quickaction.bb.database.create",
                    ];
                return new Map([
                  ["OWNER", ownerList],
                  ["DBA", dbaList],
                  ["DEVELOPER", developerList],
                ]);
              },
            },
            components: {
              content: Home,
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "403",
            name: "error.403",
            components: {
              content: () => import("../views/Page403.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "404",
            name: "error.404",
            components: {
              content: () => import("../views/Page404.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "inbox",
            name: "workspace.inbox",
            meta: { title: () => t("common.inbox") },
            components: {
              content: () => import("../views/Inbox.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "anomaly-center",
            name: "workspace.anomaly-center",
            meta: { title: () => t("anomaly-center") },
            components: {
              content: () => import("../views/AnomalyCenterDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "archive",
            name: "workspace.archive",
            meta: { title: () => t("common.archive") },
            components: {
              content: () => import("../views/Archive.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            // "u" stands for user. Strictly speaking, it's not accurate because we
            // may refer to other principal type in the future. But from the endusers'
            // perspective, they are more familiar with the "user" concept.
            // We make an exception to use a shorthand here because it's a commonly
            // accessed endpoint, and maybe in the future, we will further provide a
            // shortlink such as u/<<uid>>
            path: "u/:principalId",
            name: "workspace.profile",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const principalId = parseInt(
                  route.params.principalId as string,
                  10
                );
                return usePrincipalStore().principalById(principalId).name;
              },
            },
            components: {
              content: () => import("../views/ProfileDashboard.vue"),
              leftSidebar: () => import("../views/SettingSidebar.vue"),
            },
            props: { content: true },
          },
          {
            path: "setting",
            name: "setting",
            meta: { title: () => t("common.settings") },
            components: {
              content: () => import("../layouts/SettingLayout.vue"),
              leftSidebar: () => import("../views/SettingSidebar.vue"),
            },
            props: {
              content: true,
              leftSidebar: true,
            },
            children: [
              {
                path: "",
                name: "setting.profile",
                meta: { title: () => t("settings.sidebar.profile") },
                component: () => import("../views/ProfileDashboard.vue"),
                alias: "profile",
                props: true,
              },
              {
                path: "profile/two-factor",
                name: "setting.profile.two-factor",
                meta: {
                  title: () => t("two-factor.self"),
                },
                component: () => import("../views/TwoFactorSetup.vue"),
                props: true,
              },
              {
                path: "general",
                name: "setting.workspace.general",
                meta: { title: () => t("settings.sidebar.general") },
                component: () => import("../views/SettingWorkspaceGeneral.vue"),
                props: true,
              },
              {
                path: "agent",
                name: "setting.workspace.agent",
                meta: { title: () => t("common.agents") },
                component: () => import("../views/SettingWorkspaceAgent.vue"),
                props: true,
              },
              {
                path: "project",
                name: "setting.workspace.project",
                meta: {
                  title: () => t("common.projects"),
                  quickActionListByRole: () => {
                    return new Map([
                      ["OWNER", []],
                      ["DBA", []],
                      ["DEVELOPER", []],
                    ]);
                  },
                },
                component: () => import("../views/SettingWorkspaceProject.vue"),
                props: true,
              },
              {
                path: "member",
                name: "setting.workspace.member",
                meta: { title: () => t("settings.sidebar.members") },
                component: () => import("../views/SettingWorkspaceMember.vue"),
                props: true,
              },
              {
                path: "im-integration",
                name: "setting.workspace.im-integration",
                meta: { title: () => t("settings.sidebar.im-integration") },
                component: () =>
                  import("../views/SettingWorkspaceIMIntegration.vue"),
                props: true,
              },
              {
                path: "sso",
                name: "setting.workspace.sso",
                meta: { title: () => t("settings.sidebar.sso") },
                component: () => import("../views/SettingWorkspaceSSO.vue"),
              },
              {
                path: "sso/new",
                name: "setting.workspace.sso.create",
                meta: { title: () => t("settings.sidebar.sso") },
                component: () =>
                  import("../views/SettingWorkspaceSSODetail.vue"),
              },
              {
                path: "sso/:ssoName",
                name: "setting.workspace.sso.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const name = route.params.ssoName as string;
                    return (
                      useIdentityProviderStore().getIdentityProviderByName(name)
                        ?.title || t("settings.sidebar.sso")
                    );
                  },
                },
                component: () =>
                  import("../views/SettingWorkspaceSSODetail.vue"),
                props: true,
              },
              {
                path: "sensitive-data",
                name: "setting.workspace.sensitive-data",
                meta: { title: () => t("settings.sidebar.sensitive-data") },
                component: () =>
                  import("../views/SettingWorkspaceSensitiveData.vue"),
                props: true,
              },
              {
                path: "access-control",
                name: "setting.workspace.access-control",
                meta: { title: () => t("settings.sidebar.access-control") },
                component: () =>
                  import("../views/SettingWorkspaceAccessControl.vue"),
                props: true,
              },
              {
                path: "gitops",
                name: "setting.workspace.gitops",
                meta: { title: () => t("settings.sidebar.gitops") },
                component: () => import("../views/SettingWorkspaceVCS.vue"),
                props: true,
              },
              {
                path: "gitops/new",
                name: "setting.workspace.gitops.create",
                meta: { title: () => t("repository.add-git-provider") },
                component: () =>
                  import("../views/SettingWorkspaceVCSCreate.vue"),
                props: true,
              },
              {
                path: "gitops/:vcsSlug",
                name: "setting.workspace.gitops.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.vcsSlug as string;
                    return useVCSStore().getVCSById(idFromSlug(slug)).name;
                  },
                },
                component: () =>
                  import("../views/SettingWorkspaceVCSDetail.vue"),
                props: true,
              },
              {
                path: "subscription",
                name: "setting.workspace.subscription",
                meta: { title: () => t("settings.sidebar.subscription") },
                component: () =>
                  import("../views/SettingWorkspaceSubscription.vue"),
                props: true,
              },
              {
                path: "integration/slack",
                name: "setting.workspace.integration.slack",
                meta: { title: () => t("common.slack") },
                component: () =>
                  import("../views/SettingWorkspaceIntegrationSlack.vue"),
                props: true,
              },
              {
                path: "sql-review",
                name: "setting.workspace.sql-review",
                meta: {
                  title: () => t("sql-review.title"),
                },
                component: () =>
                  import("../views/SettingWorkspaceSQLReview.vue"),
                props: true,
              },
              {
                path: "sql-review/new",
                name: "setting.workspace.sql-review.create",
                meta: {
                  title: () => t("sql-review.create.breadcrumb"),
                },
                component: () =>
                  import("../views/SettingWorkspaceSQLReviewCreate.vue"),
                props: true,
              },
              {
                path: "sql-review/:sqlReviewPolicySlug",
                name: "setting.workspace.sql-review.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.sqlReviewPolicySlug as string;
                    return (
                      useSQLReviewStore().getReviewPolicyByEnvironmentId(
                        idFromSlug(slug)
                      )?.name ?? ""
                    );
                  },
                },
                component: () =>
                  import("../views/SettingWorkspaceSQLReviewDetail.vue"),
                props: true,
              },
              {
                path: "audit-log",
                name: "setting.workspace.audit-log",
                meta: {
                  title: () => t("settings.sidebar.audit-log"),
                },
                component: () =>
                  import("../views/SettingWorkspaceAuditLog.vue"),
                props: true,
              },
              {
                path: "debug-log",
                name: "setting.workspace.debug-log",
                meta: {
                  title: () => t("settings.sidebar.debug-log"),
                },
                component: () =>
                  import("../views/SettingWorkspaceDebugLog.vue"),
                props: true,
              },
            ],
          },
          {
            path: "issue",
            name: "workspace.issue",
            meta: {
              title: () => t("common.issues"),
            },
            components: {
              content: () => import("../views/IssueDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "environment",
            name: "workspace.environment",
            meta: {
              title: () => t("common.environments"),
              quickActionListByRole: () => {
                return new Map([
                  [
                    "OWNER",
                    [
                      "quickaction.bb.environment.create",
                      "quickaction.bb.environment.reorder",
                    ],
                  ],
                  [
                    "DBA",
                    [
                      "quickaction.bb.environment.create",
                      "quickaction.bb.environment.reorder",
                    ],
                  ],
                ]);
              },
            },
            components: {
              content: () => import("../views/EnvironmentDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "environment/:environmentSlug",
            name: "workspace.environment.detail",
            meta: {
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.environmentSlug as string;
                return useEnvironmentStore().getEnvironmentById(
                  idFromSlug(slug)
                ).name;
              },
              allowBookmark: true,
            },
            components: {
              content: () => import("../views/EnvironmentDetail.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
          {
            path: "project",
            name: "workspace.project",
            meta: {
              title: () => t("common.projects"),
              quickActionListByRole: () => {
                return new Map([
                  ["OWNER", ["quickaction.bb.project.create"]],
                  ["DBA", ["quickaction.bb.project.create"]],
                  ["DEVELOPER", ["quickaction.bb.project.create"]],
                ]);
              },
            },
            components: {
              content: () => import("../views/ProjectDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "project/:projectSlug",
            components: {
              content: () => import("../layouts/ProjectLayout.vue"),
              leftSidebar: DashboardSidebar,
            },
            meta: {
              quickActionListByRole: (route: RouteLocationNormalized) => {
                const slug = route.params.projectSlug as string;
                const project = useProjectStore().getProjectById(
                  idFromSlug(slug)
                );

                if (project.rowStatus == "NORMAL") {
                  const actionList: string[] = [];

                  const currentUser = useCurrentUser();
                  let allowAlterSchemaOrChangeData = false;
                  let allowCreateDB = false;
                  let allowTransferDB = false;
                  if (
                    hasWorkspacePermission(
                      "bb.permission.workspace.manage-instance",
                      currentUser.value.role
                    )
                  ) {
                    allowAlterSchemaOrChangeData = true;
                    allowCreateDB = true;
                    allowTransferDB = true;
                  } else {
                    const memberOfProject = project.memberList.find(
                      (m) => m.principal.id === currentUser.value.id
                    );
                    if (memberOfProject) {
                      allowAlterSchemaOrChangeData = hasProjectPermission(
                        "bb.permission.project.change-database",
                        memberOfProject.role
                      );
                      allowCreateDB = hasProjectPermission(
                        "bb.permission.project.create-database",
                        memberOfProject.role
                      );
                      allowTransferDB = hasProjectPermission(
                        "bb.permission.project.transfer-database",
                        memberOfProject.role
                      );
                    }
                  }
                  if (project.id === DEFAULT_PROJECT_ID) {
                    allowAlterSchemaOrChangeData = false;
                  }
                  if (allowAlterSchemaOrChangeData) {
                    actionList.push(
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.schema.sync"
                    );
                  }

                  if (allowCreateDB) {
                    actionList.push("quickaction.bb.database.create");
                  }
                  if (allowTransferDB) {
                    actionList.push("quickaction.bb.project.database.transfer");
                  }

                  return new Map([
                    ["OWNER", actionList],
                    ["DBA", actionList],
                    ["DEVELOPER", actionList],
                  ]);
                }
                return new Map();
              },
            },
            props: { content: true },
            children: [
              {
                path: "",
                name: "workspace.project.detail",
                meta: {
                  overrideBreadcrumb: (route: RouteLocationNormalized) => {
                    const slug = route.params.projectSlug as string;
                    const projectId = idFromSlug(slug);
                    if (projectId === DEFAULT_PROJECT_ID) {
                      return true;
                    }
                    return false;
                  },
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.projectSlug as string;
                    const projectId = idFromSlug(slug);
                    if (projectId === DEFAULT_PROJECT_ID) {
                      return t("database.unassigned-databases");
                    }
                    return useProjectStore().getProjectById(projectId).name;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/ProjectDetail.vue"),
                props: true,
              },
              {
                path: "webhook/new",
                name: "workspace.project.hook.create",
                meta: {
                  title: () => t("project.webhook.create-webhook"),
                },
                component: () => import("../views/ProjectWebhookCreate.vue"),
                props: true,
              },
              {
                path: "webhook/:projectWebhookSlug",
                name: "workspace.project.hook.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const projectSlug = route.params.projectSlug as string;
                    const projectWebhookSlug = route.params
                      .projectWebhookSlug as string;
                    return `${t("common.webhook")} - ${
                      useProjectWebhookStore().projectWebhookById(
                        idFromSlug(projectSlug),
                        idFromSlug(projectWebhookSlug)
                      ).name
                    }`;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/ProjectWebhookDetail.vue"),
                props: true,
              },
            ],
          },
          {
            path: "instance",
            name: "workspace.instance",
            meta: {
              title: () => t("common.instances"),
              quickActionListByRole: () => {
                return new Map([
                  ["OWNER", ["quickaction.bb.instance.create"]],
                  ["DBA", ["quickaction.bb.instance.create"]],
                ]);
              },
            },
            components: {
              content: () => import("../views/InstanceDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db",
            name: "workspace.database",
            meta: {
              title: () => t("common.databases"),
              quickActionListByRole: () => {
                const hasDBAWorkflowFeature = hasFeature(
                  "bb.feature.dba-workflow"
                );
                const ownerList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      // "quickaction.bb.database.troubleshoot",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                    ];
                const dbaList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      // "quickaction.bb.database.troubleshoot",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                    ];
                const developerList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      // "quickaction.bb.database.request",
                      // "quickaction.bb.database.troubleshoot",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                    ];
                return new Map([
                  ["OWNER", ownerList],
                  ["DBA", dbaList],
                  ["DEVELOPER", developerList],
                ]);
              },
            },
            components: {
              content: () => import("../views/DatabaseDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
          },
          {
            path: "db/:databaseSlug",
            components: {
              content: DatabaseLayout,
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
            children: [
              {
                path: "",
                name: "workspace.database.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.databaseSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return t("common.new");
                    }
                    return useDatabaseStore().getDatabaseById(idFromSlug(slug))
                      .name;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/DatabaseDetail.vue"),
                props: true,
              },
              {
                path: "table/:tableName",
                name: "workspace.database.table.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const schemaName = route.query.schema?.toString() || "";
                    let tableName = route.params.tableName;
                    if (schemaName) {
                      tableName = `"${schemaName}"."${tableName}"`;
                    }
                    return `${t("db.tables")} - ${tableName}`;
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/TableDetail.vue"),
                props: true,
              },
              {
                path: "history/:migrationHistorySlug",
                name: "workspace.database.history.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.migrationHistorySlug as string;
                    const migrationHistory =
                      useInstanceStore().getMigrationHistoryById(
                        migrationHistoryIdFromSlug(slug)
                      );
                    return migrationHistory?.version ?? "";
                  },
                  allowBookmark: true,
                },
                component: () => import("../views/MigrationHistoryDetail.vue"),
                props: (to) => ({
                  key: to.fullPath, // force refresh the component when slug changed
                }),
              },
            ],
          },
          {
            path: "instance/:instanceSlug",
            components: {
              content: InstanceLayout,
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
            children: [
              {
                path: "",
                name: "workspace.instance.detail",
                meta: {
                  title: (route: RouteLocationNormalized) => {
                    const slug = route.params.instanceSlug as string;
                    if (slug.toLowerCase() == "new") {
                      return t("common.new");
                    }
                    return useInstanceStore().getInstanceById(idFromSlug(slug))
                      .name;
                  },
                },
                component: () => import("../views/InstanceDetail.vue"),
                props: true,
              },
            ],
          },
          {
            path: "issue/:issueSlug",
            name: "workspace.issue.detail",
            meta: {
              allowBookmark: true,
              overrideTitle: true,
            },
            components: {
              content: () => import("../views/IssueDetail.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true },
          },
        ],
      },
    ],
  },
  {
    path: "/sql-editor",
    name: "sql-editor",
    component: SQLEditorLayout,
    children: [
      {
        path: "",
        name: SQL_EDITOR_HOME_MODULE,
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
        props: true,
      },
      {
        path: "/sql-editor/:connectionSlug",
        name: "sql-editor.detail",
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
        props: true,
      },
      {
        path: "/sql-editor/sheet/:sheetSlug",
        name: "sql-editor.share",
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditorPage.vue"),
        props: true,
      },
    ],
  },
  {
    path: "/sheets",
    name: "sheets",
    component: SheetDashboardLayout,
    children: [
      {
        path: "",
        name: "sheets.dashboard",
        meta: { title: () => "Sheets" },
        component: () => import("../views/SheetDashboard.vue"),
      },
      {
        path: "my",
        name: "sheets.my",
        meta: { title: () => "Sheets" },
        component: () => import("../views/SheetDashboard.vue"),
      },
      {
        path: "shared",
        name: "sheets.shared",
        meta: { title: () => "Sheets" },
        component: () => import("../views/SheetDashboard.vue"),
      },
      {
        path: "starred",
        name: "sheets.starred",
        meta: { title: () => "Sheets" },
        component: () => import("../views/SheetDashboard.vue"),
      },
    ],
  },
];

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
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

  const authStore = useAuthStore();
  const databaseStore = useDatabaseStore();
  const dbSchemaStore = useDBSchemaStore();
  const environmentStore = useEnvironmentStore();
  const instanceStore = useInstanceStore();
  const routerStore = useRouterStore();
  const projectWebhookStore = useProjectWebhookStore();
  const projectStore = useProjectStore();

  const isLoggedIn = authStore.isLoggedIn();

  const fromModule = from.name
    ? from.name.toString().split(".")[0]
    : HOME_MODULE;
  const toModule = to.name ? to.name.toString().split(".")[0] : HOME_MODULE;

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

  if (
    to.name === SIGNIN_MODULE ||
    to.name === SIGNUP_MODULE ||
    to.name === ACTIVATE_MODULE ||
    to.name === PASSWORD_RESET_MODULE ||
    to.name === PASSWORD_FORGOT_MODULE
  ) {
    useTabStore().reset();
    if (isLoggedIn) {
      if (typeof to.query.redirect === "string") {
        location.replace(to.query.redirect);
        return;
      }
      next({ name: HOME_MODULE, replace: true });
    } else {
      if (to.name === ACTIVATE_MODULE) {
        const token = to.query.token;
        if (token) {
          // TODO(tianzhou): Needs to validate the activate token
          next();
        } else {
          // Go to signup if token is missing
          next({ name: SIGNUP_MODULE, replace: true });
        }
      } else {
        next();
      }
    }
    return;
  } else {
    if (!isLoggedIn) {
      const query: any = {};
      if (to.fullPath !== "/") {
        query["redirect"] = to.fullPath;
      }

      next({
        name: SIGNIN_MODULE,
        query: query,
        replace: true,
      });
      return;
    }
  }

  // If there is a `redirect` in query param and prev page is signin or signup, redirect to the target route
  if (
    (from.name === SIGNIN_MODULE || from.name === SIGNUP_MODULE) &&
    typeof from.query.redirect === "string"
  ) {
    window.location.href = from.query.redirect;
    return;
  }

  if (to.name === SQL_EDITOR_HOME_MODULE) {
    const onboardingStateStore = useOnboardingStateStore();
    if (onboardingStateStore.getStateByKey("sql-editor")) {
      // Open the "Sample Sheet" when the first time onboarding SQL Editor
      onboardingStateStore.consume("sql-editor");
      next({
        path: `/sql-editor/sheet/sample-sheet-101`,
        replace: true,
      });
      return;
    }
  }

  const currentUser = useCurrentUser();

  if (to.name?.toString().startsWith("setting.workspace.im-integration")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-im-integration",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.sso")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-sso",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.gitops")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-vcs-provider",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.project")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-project",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.audit-log")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.audit-log",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.debug-log")) {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.debug-log",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name === "workspace.instance") {
    if (
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-instance",
        currentUser.value.role
      )
    ) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("workspace.database.datasource")) {
    if (
      !hasFeature("bb.feature.data-source") ||
      !hasWorkspacePermission(
        "bb.permission.workspace.manage-instance",
        currentUser.value.role
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
    to.name === "workspace.home" ||
    to.name === "workspace.inbox" ||
    to.name === "workspace.anomaly-center" ||
    to.name === "workspace.project" ||
    to.name === "workspace.instance" ||
    to.name === "workspace.database" ||
    to.name === "workspace.archive" ||
    to.name === "workspace.issue" ||
    to.name === "workspace.environment" ||
    to.name === "sql-editor.home" ||
    to.name?.toString().startsWith("sheets") ||
    (to.name?.toString().startsWith("setting") &&
      to.name?.toString() != "setting.workspace.gitops.detail" &&
      to.name?.toString() != "setting.workspace.sql-review.detail")
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
  const principalId = routerSlug.principalId;
  const environmentSlug = routerSlug.environmentSlug;
  const projectSlug = routerSlug.projectSlug;
  const projectWebhookSlug = routerSlug.projectWebhookSlug;
  const issueSlug = routerSlug.issueSlug;
  const instanceSlug = routerSlug.instanceSlug;
  const databaseSlug = routerSlug.databaseSlug;
  const dataSourceSlug = routerSlug.dataSourceSlug;
  const migrationHistorySlug = routerSlug.migrationHistorySlug;
  const vcsSlug = routerSlug.vcsSlug;
  const connectionSlug = routerSlug.connectionSlug;
  const sheetSlug = routerSlug.sheetSlug;
  const sqlReviewPolicySlug = routerSlug.sqlReviewPolicySlug;

  if (principalId) {
    usePrincipalStore()
      .fetchPrincipalById(principalId)
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
    const env = environmentStore.getEnvironmentById(
      idFromSlug(environmentSlug)
    );
    // getEnvironmentById returns unknown("ENVIRONMENT") when it doesn't exist
    // so we need to check the id here
    if (env && env.id !== UNKNOWN_ID) {
      next();
      return;
    }
    next({
      name: "error.404",
      replace: false,
    });
  }

  if (projectSlug) {
    projectStore
      .fetchProjectById(idFromSlug(projectSlug))
      .then(() => {
        if (!projectWebhookSlug) {
          next();
        } else {
          projectWebhookStore
            .fetchProjectWebhookById({
              projectId: idFromSlug(projectSlug),
              projectWebhookId: idFromSlug(projectWebhookSlug),
            })
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

  if (databaseSlug) {
    if (databaseSlug.toLowerCase() == "grant") {
      next();
      return;
    }
    databaseStore
      .fetchDatabaseById(idFromSlug(databaseSlug))
      .then((database: Database) => {
        dbSchemaStore
          .getOrFetchDatabaseMetadataById(database.id, true)
          .then(() => {
            if (!dataSourceSlug && !migrationHistorySlug) {
              next();
            } else if (dataSourceSlug) {
              useDataSourceStore()
                .fetchDataSourceById({
                  dataSourceId: idFromSlug(dataSourceSlug),
                  databaseId: database.id,
                })
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
            } else if (migrationHistorySlug) {
              instanceStore
                .fetchMigrationHistoryById({
                  instanceId: database.instance.id,
                  migrationHistoryId:
                    migrationHistoryIdFromSlug(migrationHistorySlug),
                })
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
            }
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
    instanceStore
      .fetchInstanceById(idFromSlug(instanceSlug))
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
    useVCSStore()
      .fetchVCSById(idFromSlug(vcsSlug))
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
      .fetchReviewPolicyByEnvironmentId(idFromSlug(sqlReviewPolicySlug))
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
    const sheetId = idFromSlug(sheetSlug);
    useSheetStore()
      .fetchSheetById(sheetId)
      .then(() => next())
      .catch(() => next());
    return;
  }

  if (connectionSlug) {
    const [instanceSlug, databaseSlug = ""] = connectionSlug.split("_");
    const instanceId = idFromSlug(instanceSlug);
    const databaseId = idFromSlug(databaseSlug);
    if (Number.isNaN(databaseId)) {
      // Connected to instance
      useConnectionTreeStore()
        .fetchConnectionByInstanceId(instanceId)
        .then(() => next())
        .catch(() => next());
    } else {
      // Connected to db
      useConnectionTreeStore()
        .fetchConnectionByInstanceIdAndDatabaseId(instanceId, databaseId)
        .then(() => next())
        .catch(() => next());
    }
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
