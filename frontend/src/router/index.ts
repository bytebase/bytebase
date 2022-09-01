import { nextTick } from "vue";
import {
  createRouter,
  createWebHistory,
  RouteLocationNormalized,
  RouteRecordRaw,
} from "vue-router";
import BodyLayout from "../layouts/BodyLayout.vue";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import DatabaseLayout from "../layouts/DatabaseLayout.vue";
import InstanceLayout from "../layouts/InstanceLayout.vue";
import SplashLayout from "../layouts/SplashLayout.vue";
import SQLEditorLayout from "../layouts/SQLEditorLayout.vue";
import SheetDashboardLayout from "../layouts/SheetDashboardLayout.vue";
import { t } from "../plugins/i18n";
import {
  useAuthStore,
  useDatabaseStore,
  useEnvironmentStore,
  useInstanceStore,
  useIssueStore,
  usePrincipalStore,
  useRouterStore,
  useSubscriptionStore,
} from "../store";
import {
  Database,
  DEFAULT_PROJECT_ID,
  PlanType,
  QuickActionType,
  Sheet,
  UNKNOWN_ID,
} from "../types";
import { idFromSlug, isDBAOrOwner, isOwner, isProjectOwner } from "../utils";
// import PasswordReset from "../views/auth/PasswordReset.vue";
import Signin from "../views/auth/Signin.vue";
import Signup from "../views/auth/Signup.vue";
import DashboardSidebar from "../views/DashboardSidebar.vue";
import Home from "../views/Home.vue";
import {
  useTabStore,
  hasFeature,
  useVCSStore,
  useProjectWebhookStore,
  useDataSourceStore,
  useSQLReviewStore,
  useProjectStore,
  useTableStore,
  useSQLEditorStore,
  useSheetStore,
} from "@/store";

const HOME_MODULE = "workspace.home";
const AUTH_MODULE = "auth";
const SIGNIN_MODULE = "auth.signin";
const SIGNUP_MODULE = "auth.signup";
const ACTIVATE_MODULE = "auth.activate";
const PASSWORD_RESET_MODULE = "auth.password.reset";
const PASSWORD_FORGOT_MODULE = "auth.password.forgot";

// console.log(useProjectWebhookStore());

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
                      "quickaction.bb.database.create",
                      "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                      "quickaction.bb.user.manage",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                      "quickaction.bb.user.manage",
                    ];
                const dbaList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.instance.create",
                      "quickaction.bb.project.create",
                    ];
                const developerList: QuickActionType[] = hasDBAWorkflowFeature
                  ? [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      // "quickaction.bb.database.request",
                      // "quickaction.bb.database.troubleshoot",
                      "quickaction.bb.project.create",
                    ]
                  : [
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update",
                      "quickaction.bb.database.create",
                      "quickaction.bb.project.create",
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
              leftSidebar: DashboardSidebar,
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
                path: "general",
                name: "setting.workspace.general",
                meta: { title: () => t("settings.sidebar.general") },
                component: () => import("../views/SettingWorkspaceGeneral.vue"),
                props: true,
              },
              {
                path: "label",
                name: "setting.workspace.label",
                meta: { title: () => t("settings.sidebar.labels") },
                component: () => import("../views/SettingWorkspaceLabel.vue"),
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
                path: "version-control",
                name: "setting.workspace.version-control",
                meta: { title: () => t("settings.sidebar.version-control") },
                component: () => import("../views/SettingWorkspaceVCS.vue"),
                props: true,
              },
              {
                path: "version-control/new",
                name: "setting.workspace.version-control.create",
                meta: { title: () => t("repository.add-git-provider") },
                component: () =>
                  import("../views/SettingWorkspaceVCSCreate.vue"),
                props: true,
              },
              {
                path: "version-control/:vcsSlug",
                name: "setting.workspace.version-control.detail",
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
                path: "billing",
                name: "setting.workspace.billing",
                meta: { title: () => t("common.billings") },
                component: () => import("../views/SettingWorkspaceBilling.vue"),
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

                  const currentUser = useAuthStore().currentUser;
                  let allowAlterSchemaOrChangeData = false;
                  let allowCreateOrTransferDB = false;
                  if (isDBAOrOwner(currentUser.role)) {
                    // Yes to workspace owner and DBA
                    allowAlterSchemaOrChangeData = true;
                    allowCreateOrTransferDB = true;
                  } else {
                    const memberOfProject = project.memberList.find(
                      (m) => m.principal.id === currentUser.id
                    );
                    if (memberOfProject) {
                      // If current user is a member of this project
                      // we are allowed to alter schema and change data.
                      allowAlterSchemaOrChangeData = true;

                      const plan = useSubscriptionStore().currentPlan;
                      allowCreateOrTransferDB =
                        plan === PlanType.ENTERPRISE
                          ? // For ENTERPRISE plan, only
                            //   - workspace owner and DBA
                            //   - developers as the project owner
                            // can create/transfer DB.
                            // Other developers are not allowed.
                            isProjectOwner(memberOfProject.role)
                          : // For TEAM plan, all members of the project are allowed
                            true;
                    }
                  }
                  if (allowAlterSchemaOrChangeData) {
                    actionList.push(
                      "quickaction.bb.database.schema.update",
                      "quickaction.bb.database.data.update"
                    );
                  }

                  if (allowCreateOrTransferDB) {
                    actionList.push(
                      "quickaction.bb.database.create",
                      "quickaction.bb.project.database.transfer"
                    );
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
                      "quickaction.bb.database.troubleshoot",
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
                      "quickaction.bb.database.troubleshoot",
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
                    return `${t("db.tables")} - ${route.params.tableName}`;
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
                        idFromSlug(slug)
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
              title: (route: RouteLocationNormalized) => {
                const slug = route.params.issueSlug as string;
                if (slug.toLowerCase() == "new") {
                  return t("issue.new-issue");
                }
                const issue = useIssueStore().getIssueById(idFromSlug(slug));
                return issue.name;
              },
              allowBookmark: true,
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
        name: "sql-editor.home",
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditor.vue"),
        props: true,
      },
      {
        path: "/sql-editor/:connectionSlug",
        name: "sql-editor.detail",
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditor.vue"),
        props: true,
      },
      {
        path: "/sql-editor/:connectionSlug/:sheetSlug",
        name: "sql-editor.share",
        meta: { title: () => "SQL Editor" },
        component: () => import("../views/sql-editor/SQLEditor.vue"),
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
  const environmentStore = useEnvironmentStore();
  const instanceStore = useInstanceStore();
  const issueStore = useIssueStore();
  const tabStore = useTabStore();
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

  // OAuth callback route is a relay to receive the OAuth callback and dispatch the corresponding OAuth event. It's called in the following scenarios:
  // - Login via OAuth
  // - Setup VCS provider
  // - Setup GitOps workflow in a project
  if (to.name === "oauth-callback") {
    next();
  }

  if (
    to.name === SIGNIN_MODULE ||
    to.name === SIGNUP_MODULE ||
    to.name === ACTIVATE_MODULE ||
    to.name === PASSWORD_RESET_MODULE ||
    to.name === PASSWORD_FORGOT_MODULE
  ) {
    if (isLoggedIn) {
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

  const currentUser = authStore.currentUser;

  if (to.name?.toString().startsWith("setting.workspace.version-control")) {
    // Returns 403 immediately if not Owner. Otherwise, we may need to fetch the VCS detail
    if (!isOwner(currentUser.role)) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.project")) {
    // Returns 403 immediately if not DBA or Owner.
    if (!isDBAOrOwner(currentUser.role)) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name?.toString().startsWith("setting.workspace.debug-log")) {
    // Returns 403 immediately if not DBA or Owner.
    if (!isDBAOrOwner(currentUser.role)) {
      next({
        name: "error.403",
        replace: false,
      });
      return;
    }
  }

  if (to.name === "workspace.instance") {
    if (
      !hasFeature("bb.feature.dba-workflow") ||
      isDBAOrOwner(currentUser.role)
    ) {
      next();
    } else {
      next({
        name: "error.403",
        replace: false,
      });
    }
    return;
  }

  if (to.name?.toString().startsWith("workspace.database.datasource")) {
    if (
      !hasFeature("bb.feature.data-source") ||
      !isDBAOrOwner(currentUser.role)
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
    to.name === "workspace.database" ||
    to.name === "workspace.archive" ||
    to.name === "workspace.issue" ||
    to.name === "workspace.environment" ||
    to.name === "sql-editor.home" ||
    to.name?.toString().startsWith("sheets") ||
    (to.name?.toString().startsWith("setting") &&
      to.name?.toString() != "setting.workspace.version-control.detail" &&
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
  const tableName = routerSlug.tableName;
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
    if (issueSlug.toLowerCase() == "new") {
      // For preparing the database if user visits creating issue url directly.
      const requests: Promise<any>[] = [];
      if (to.query.databaseList) {
        for (const databaseId of (to.query.databaseList as string).split(",")) {
          requests.push(
            databaseStore.fetchDatabaseById(parseInt(databaseId, 10))
          );
        }
      }
      Promise.all(requests)
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
    issueStore
      .fetchIssueById(idFromSlug(issueSlug))
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

  if (databaseSlug) {
    if (databaseSlug.toLowerCase() == "grant") {
      next();
      return;
    }
    databaseStore
      .fetchDatabaseById(idFromSlug(databaseSlug))
      .then((database: Database) => {
        if (!tableName && !dataSourceSlug && !migrationHistorySlug) {
          next();
        } else if (tableName) {
          useTableStore()
            .fetchTableByDatabaseIdAndTableName({
              databaseId: database.id,
              tableName,
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
              migrationHistoryId: idFromSlug(migrationHistorySlug),
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

  if (connectionSlug) {
    const [, instanceId, , databaseId] = connectionSlug.split("_");
    useSQLEditorStore()
      .fetchConnectionByInstanceIdAndDatabaseId({
        instanceId: Number(instanceId),
        databaseId: Number(databaseId),
      })
      .then(() => {
        // for sharing the sheet to others
        if (sheetSlug) {
          const [_, sheetId] = sheetSlug.split("_");
          useSheetStore()
            .fetchSheetById(Number(sheetId))
            .then((sheet: Sheet) => {
              tabStore.addTab({
                name: sheet.name,
                statement: sheet.statement,
                isSaved: true,
              });
              tabStore.updateCurrentTab({
                sheetId: sheet.id,
              });
              useSQLEditorStore().setSQLEditorState({
                sharedSheet: sheet,
              });

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

  next({
    name: "error.404",
    replace: false,
  });
});

router.afterEach((to /*, from */) => {
  // Needs to use nextTick otherwise title will still be the one from the previous route.
  nextTick(() => {
    if (to.meta.title) {
      document.title = to.meta.title(to);
    } else {
      document.title = "Bytebase";
    }
  });
});
