import { nextTick, ref } from "vue";
import {
  createRouter,
  createWebHistory,
  RouteLocationNormalized,
  RouteRecordRaw,
} from "vue-router";
import { useTitle } from "@vueuse/core";
import { pull, startCase } from "lodash-es";

import BodyLayout from "@/layouts/BodyLayout.vue";
import DashboardLayout from "@/layouts/DashboardLayout.vue";
import DatabaseLayout from "@/layouts/DatabaseLayout.vue";
import InstanceLayout from "@/layouts/InstanceLayout.vue";
import SplashLayout from "@/layouts/SplashLayout.vue";
import SQLEditorLayout from "@/layouts/SQLEditorLayout.vue";
import SheetDashboardLayout from "@/layouts/SheetDashboardLayout.vue";
import { t } from "@/plugins/i18n";
import {
  DEFAULT_PROJECT_ID,
  DEFAULT_PROJECT_V1_NAME,
  QuickActionType,
  unknownUser,
  UNKNOWN_ID,
} from "@/types";
import {
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
  idFromSlug,
  sheetNameFromSlug,
  isOwnerOfProjectV1,
  extractChangeHistoryUID,
} from "@/utils";
import Signin from "@/views/auth/Signin.vue";
import Signup from "@/views/auth/Signup.vue";
import MultiFactor from "@/views/auth/MultiFactor.vue";
import DashboardSidebar from "@/views/DashboardSidebar.vue";
import Home from "@/views/Home.vue";
import {
  hasFeature,
  useVCSV1Store,
  useDataSourceStore,
  useSQLReviewStore,
  useSheetV1Store,
  useAuthStore,
  useActuatorV1Store,
  useLegacyInstanceStore,
  useRouterStore,
  useDBSchemaV1Store,
  useConnectionTreeStore,
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
} from "@/store";
import { useConversationStore } from "@/plugins/ai/store";
import { State } from "@/types/proto/v1/common";

const HOME_MODULE = "workspace.home";
const AUTH_MODULE = "auth";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const MFA_MODULE = "auth.mfa";
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
    path: "/auth/mfa",
    name: MFA_MODULE,
    meta: { title: () => t("multi-factor.self") },
    component: MultiFactor,
    props: true,
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
    path: "/2fa/setup",
    name: "2fa.setup",
    meta: {
      title: () => t("two-factor.self"),
    },
    component: () => import("../views/TwoFactorRequired.vue"),
    props: true,
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
                const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
                  "quickaction.bb.database.schema.update",
                  "quickaction.bb.database.schema.design",
                  "quickaction.bb.database.data.update",
                  "quickaction.bb.database.create",
                  "quickaction.bb.instance.create",
                ];
                const DEVELOPER_QUICK_ACTION_LIST: QuickActionType[] = [
                  "quickaction.bb.database.schema.update",
                  "quickaction.bb.database.schema.design",
                  "quickaction.bb.database.data.update",
                  "quickaction.bb.database.create",
                  "quickaction.bb.issue.grant.request.querier",
                  "quickaction.bb.issue.grant.request.exporter",
                ];
                if (hasFeature("bb.feature.dba-workflow")) {
                  pull(
                    DEVELOPER_QUICK_ACTION_LIST,
                    "quickaction.bb.database.create"
                  );
                }
                return new Map([
                  ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
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
            path: "slow-query",
            name: "workspace.slow-query",
            meta: { title: () => startCase(t("slow-query.slow-queries")) },
            components: {
              content: () =>
                import("../views/SlowQuery/SlowQueryDashboard.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "sync-schema",
            name: "workspace.sync-schema",
            meta: { title: () => startCase(t("database.sync-schema.title")) },
            components: {
              content: () => import("../views/SyncDatabaseSchema/index.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: {
              content: true,
              leftSidebar: true,
            },
          },
          {
            path: "export-center",
            name: "workspace.export-center",
            meta: { title: () => startCase(t("export-center.self")) },
            components: {
              content: () => import("../views/ExportCenter/index.vue"),
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
                const userUID = route.params.principalId as string;
                const user =
                  useUserStore().getUserById(userUID) ?? unknownUser();
                return user.title;
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
                path: "role",
                name: "setting.workspace.role",
                meta: { title: () => t("settings.sidebar.custom-roles") },
                component: () => import("../views/SettingWorkspaceRole.vue"),
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
                path: "risk-center",
                name: "setting.workspace.risk-center",
                meta: { title: () => t("custom-approval.risk.risk-center") },
                component: () =>
                  import("../views/SettingWorkspaceRiskCenter.vue"),
                props: true,
              },
              {
                path: "custom-approval",
                name: "setting.workspace.custom-approval",
                meta: { title: () => t("custom-approval.self") },
                component: () =>
                  import("../views/SettingWorkspaceCustomApproval.vue"),
                props: true,
              },
              {
                path: "slow-query",
                name: "setting.workspace.slow-query",
                meta: { title: () => startCase(t("slow-query.self")) },
                component: () =>
                  import("../views/SettingWorkspaceSlowQuery.vue"),
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
                    return (
                      useVCSV1Store().getVCSByUid(idFromSlug(slug))?.title ?? ""
                    );
                  },
                },
                component: () =>
                  import("../views/SettingWorkspaceVCSDetail.vue"),
                props: true,
              },
              {
                path: "mail-delivery",
                name: "setting.workspace.mail-delivery",
                meta: { title: () => t("settings.sidebar.mail-delivery") },
                component: () =>
                  import("../views/SettingWorkspaceMailDelivery.vue"),
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
                      useSQLReviewStore().getReviewPolicyByEnvironmentUID(
                        String(idFromSlug(slug))
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
                const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
                  "quickaction.bb.environment.create",
                  "quickaction.bb.environment.reorder",
                ];
                return new Map([
                  ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DEVELOPER", []],
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
                return useEnvironmentV1Store().getEnvironmentByUID(
                  String(idFromSlug(slug))
                ).title;
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
              quickActionListByRole: (route) => {
                const slug = route.params.projectSlug as string;
                const project = useProjectV1Store().getProjectByUID(
                  String(idFromSlug(slug))
                );

                if (project.state === State.ACTIVE) {
                  const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
                    "quickaction.bb.database.schema.update",
                    "quickaction.bb.database.schema.design",
                    "quickaction.bb.database.data.update",
                    "quickaction.bb.database.create",
                    "quickaction.bb.project.database.transfer",
                    "quickaction.bb.project.database.transfer-out",
                  ];
                  const DEVELOPER_QUICK_ACTION_LIST: QuickActionType[] = [];

                  const currentUserV1 = useCurrentUserV1();
                  if (
                    project.name !== DEFAULT_PROJECT_V1_NAME &&
                    hasPermissionInProjectV1(
                      project.iamPolicy,
                      currentUserV1.value,
                      "bb.permission.project.change-database"
                    )
                  ) {
                    // Default project (Unassigned databases) are not allowed
                    // to be changed.
                    DEVELOPER_QUICK_ACTION_LIST.push(
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.schema.design",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create"
                    );
                  }
                  if (
                    hasPermissionInProjectV1(
                      project.iamPolicy,
                      currentUserV1.value,
                      "bb.permission.project.transfer-database"
                    )
                  ) {
                    DEVELOPER_QUICK_ACTION_LIST.push(
                      "quickaction.bb.project.database.transfer",
                      "quickaction.bb.project.database.transfer-out"
                    );
                  }
                  if (
                    !isOwnerOfProjectV1(project.iamPolicy, currentUserV1.value)
                  ) {
                    DEVELOPER_QUICK_ACTION_LIST.push(
                      "quickaction.bb.issue.grant.request.querier",
                      "quickaction.bb.issue.grant.request.exporter"
                    );
                  }

                  if (hasFeature("bb.feature.dba-workflow")) {
                    pull(
                      DEVELOPER_QUICK_ACTION_LIST,
                      "quickaction.bb.database.create"
                    );
                  }

                  return new Map([
                    ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
                    ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
                    ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
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
                    const projectV1 = useProjectV1Store().getProjectByUID(
                      String(projectId)
                    );
                    return projectV1.title;
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
                    const project = useProjectV1Store().getProjectByUID(
                      String(idFromSlug(projectSlug))
                    );
                    const webhook =
                      useProjectWebhookV1Store().getProjectWebhookFromProjectById(
                        project,
                        idFromSlug(projectWebhookSlug)
                      );

                    return `${t("common.webhook")} - ${
                      webhook?.title ?? "unknown"
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
                  [
                    "OWNER",
                    [
                      "quickaction.bb.instance.create",
                      "quickaction.bb.subscription.license-assignment",
                    ],
                  ],
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
                const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
                  "quickaction.bb.database.schema.update",
                  "quickaction.bb.database.schema.design",
                  "quickaction.bb.database.data.update",
                  "quickaction.bb.database.create",
                ];
                const DEVELOPER_QUICK_ACTION_LIST: QuickActionType[] = [
                  "quickaction.bb.database.schema.update",
                  "quickaction.bb.database.schema.design",
                  "quickaction.bb.database.data.update",
                  "quickaction.bb.database.create",
                  "quickaction.bb.issue.grant.request.querier",
                  "quickaction.bb.issue.grant.request.exporter",
                ];

                if (hasFeature("bb.feature.dba-workflow")) {
                  pull(
                    DEVELOPER_QUICK_ACTION_LIST,
                    "quickaction.bb.database.create"
                  );
                }

                return new Map([
                  ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
                  ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
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
                    return useDatabaseV1Store().getDatabaseByUID(
                      String(idFromSlug(slug))
                    ).databaseName;
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
            ],
          },
          {
            path: "instances/:instance/databases/:database/changeHistories/:changeHistorySlug",
            name: "workspace.database.history.detail",
            meta: {
              title: (route) => {
                const parent = `instances/${route.params.instance}/databases/${route.params.database}`;
                const uid = extractChangeHistoryUID(
                  route.params.changeHistorySlug as string
                );
                const name = `${parent}/changeHistories/${uid}`;
                const history =
                  useChangeHistoryStore().getChangeHistoryByName(name);

                return history?.version ?? "";
              },
            },
            components: {
              content: () => import("../views/ChangeHistoryDetail.vue"),
              leftSidebar: DashboardSidebar,
            },
            props: { content: true, leftSidebar: true },
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
                    return useInstanceV1Store().getInstanceByUID(
                      String(idFromSlug(slug))
                    ).title;
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
          // Resource name related routes.
          {
            path: "projects/:projectName",
            children: [
              {
                path: "database-groups/:databaseGroupName",
                name: "workspace.database-group.detail",
                components: {
                  content: () => import("../views/DatabaseGroupDetail.vue"),
                  leftSidebar: DashboardSidebar,
                },
                props: true,
              },
              {
                path: "database-groups/:databaseGroupName/table-groups/:schemaGroupName",
                name: "workspace.database-group.table-group.detail",
                components: {
                  content: () => import("../views/SchemaGroupDetail.vue"),
                  leftSidebar: DashboardSidebar,
                },
                props: true,
              },
            ],
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
  const dbSchemaStore = useDBSchemaV1Store();
  const instanceStore = useLegacyInstanceStore();
  const routerStore = useRouterStore();
  const projectV1Store = useProjectV1Store();
  const projectWebhookV1Store = useProjectWebhookV1Store();

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
    to.name === MFA_MODULE ||
    to.name === ACTIVATE_MODULE ||
    to.name === PASSWORD_RESET_MODULE ||
    to.name === PASSWORD_FORGOT_MODULE
  ) {
    useTabStore().reset();
    useConversationStore().reset();
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

  if (to.name === "2fa.setup") {
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
        name: "2fa.setup",
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

  if (to.name?.toString().startsWith("setting.workspace.im-integration")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-im-integration",
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

  if (to.name?.toString().startsWith("setting.workspace.sso")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-sso",
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

  if (to.name?.toString().startsWith("setting.workspace.gitops")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-vcs-provider",
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

  if (to.name?.toString().startsWith("setting.workspace.project")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-project",
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

  if (to.name?.toString().startsWith("setting.workspace.audit-log")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.audit-log",
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

  if (to.name?.toString().startsWith("setting.workspace.debug-log")) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.debug-log",
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
    to.name === "workspace.home" ||
    to.name === "workspace.inbox" ||
    to.name === "workspace.slow-query" ||
    to.name === "workspace.sync-schema" ||
    to.name === "workspace.export-center" ||
    to.name === "workspace.anomaly-center" ||
    to.name === "workspace.project" ||
    to.name === "workspace.instance" ||
    to.name === "workspace.database" ||
    to.name === "workspace.archive" ||
    to.name === "workspace.issue" ||
    to.name === "workspace.environment" ||
    to.name === "sql-editor.home" ||
    to.name?.toString().startsWith("workspace.database-group") ||
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

  if (to.name === "workspace.database.history.detail") {
    const parent = `instances/${to.params.instance}/databases/${to.params.database}`;
    const uid = extractChangeHistoryUID(to.params.changeHistorySlug as string);
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
  const principalId = routerSlug.principalId;
  const environmentSlug = routerSlug.environmentSlug;
  const projectSlug = routerSlug.projectSlug;
  const projectWebhookSlug = routerSlug.projectWebhookSlug;
  const issueSlug = routerSlug.issueSlug;
  const instanceSlug = routerSlug.instanceSlug;
  const databaseSlug = routerSlug.databaseSlug;
  const dataSourceSlug = routerSlug.dataSourceSlug;
  const vcsSlug = routerSlug.vcsSlug;
  const connectionSlug = routerSlug.connectionSlug;
  const sheetSlug = routerSlug.sheetSlug;
  const sqlReviewPolicySlug = routerSlug.sqlReviewPolicySlug;

  if (principalId) {
    useUserStore()
      .getOrFetchUserById(String(principalId))
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

  if (projectSlug) {
    projectV1Store
      .fetchProjectByUID(String(idFromSlug(projectSlug)))
      .then((project) => {
        if (!projectWebhookSlug) {
          next();
        } else {
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
    useDatabaseV1Store()
      .fetchDatabaseByUID(String(idFromSlug(databaseSlug)))
      .then((database) => {
        dbSchemaStore
          .getOrFetchDatabaseMetadata(database.name, true)
          .then(() => {
            if (!dataSourceSlug) {
              next();
            } else if (dataSourceSlug) {
              useDataSourceStore()
                .fetchDataSourceById({
                  dataSourceId: idFromSlug(dataSourceSlug),
                  databaseId: Number(database.uid),
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
    instanceStore;
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

  if (connectionSlug) {
    const [instanceSlug, databaseSlug = ""] = connectionSlug.split("_");
    const instanceId = idFromSlug(instanceSlug);
    const databaseId = idFromSlug(databaseSlug);
    if (Number.isNaN(databaseId)) {
      // Connected to instance
      useConnectionTreeStore()
        .fetchConnectionByInstanceId(String(instanceId))
        .then(() => next())
        .catch(() => next());
    } else {
      // Connected to db
      useConnectionTreeStore()
        .fetchConnectionByInstanceIdAndDatabaseId(
          String(instanceId),
          String(databaseId)
        )
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
