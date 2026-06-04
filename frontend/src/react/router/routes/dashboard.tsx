import { Navigate, type RouteObject, redirect } from "react-router-dom";
import { BodyLayout } from "@/react/app/layouts/BodyLayout";
import { DashboardLayout } from "@/react/app/layouts/DashboardLayout";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_CREATE,
  INSTANCE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DATABASE_DETAIL,
  INSTANCE_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_ACCESS_GRANTS,
  PROJECT_V1_ROUTE_AUDIT_LOGS,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATA_EXPORT,
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION,
  PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
  PROJECT_V1_ROUTE_MEMBERS,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  PROJECT_V1_ROUTE_PLANS,
  PROJECT_V1_ROUTE_RELEASE_DETAIL,
  PROJECT_V1_ROUTE_RELEASES,
  PROJECT_V1_ROUTE_SERVICE_ACCOUNTS,
  PROJECT_V1_ROUTE_SETTINGS,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
  PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES,
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_PROFILE_TWO_FACTOR,
  SETTING_ROUTE_WORKSPACE,
  SETTING_ROUTE_WORKSPACE_GENERAL,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
  WORKSPACE_ROUTE_AUDIT_LOG,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MCP,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_MY_ISSUES,
  WORKSPACE_ROUTE_RISK_ASSESSMENT,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
  WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
} from "@/react/router/handles";
import { RouteShellOutletPlaceholder } from "@/react/router/layoutPlaceholders";
import { lazyPage } from "@/react/router/lazyPage";

// Translated from `@/router/dashboard/**`. Vue named views (`content`,
// `leftSidebar`, `body`) collapse to the single `<Outlet/>` of each layout
// route. The Vue route-shell parents (`SettingRouteShell` /
// `ProjectRouteShell` / `InstanceRouteShell` / `IssuesRouteShell`) render an
// empty teleport target and require ReactRouteShellBridge props; they do not
// render an `<Outlet/>`, so each shell-parent route here uses
// `RouteShellOutletPlaceholder` to keep child leaves rendering until a later
// phase swaps in `<Outlet/>`-aware shells. Leaf routes lazy-import their page
// component.

// Workspace-level pages (the BodyLayout children).
const workspaceLevelRoutes: RouteObject[] = [
  {
    index: true,
    // The bare workspace root ("/") has no page; `rootGuard` always redirects
    // it dynamically (change-mode → SQL Editor, else last visit / landing), so
    // it intentionally renders nothing. `guardRedirect` records that for the
    // route-reachability test.
    handle: { name: WORKSPACE_ROOT_MODULE, guardRedirect: true },
  },
  {
    path: "landing",
    handle: { name: WORKSPACE_ROUTE_LANDING },
    lazy: lazyPage(
      () => import("@/react/pages/settings/LandingPage"),
      (m) => m.LandingPage
    ),
  },
  {
    path: "projects",
    handle: { name: PROJECT_V1_ROUTE_DASHBOARD },
    lazy: lazyPage(
      () => import("@/react/pages/settings/ProjectsPage"),
      (m) => m.ProjectsPage
    ),
  },
  {
    path: "instances",
    handle: { name: INSTANCE_ROUTE_DASHBOARD },
    lazy: lazyPage(
      () => import("@/react/pages/settings/InstancesPage"),
      (m) => m.InstancesPage
    ),
  },
  {
    path: "databases",
    handle: { name: DATABASE_ROUTE_DASHBOARD },
    lazy: lazyPage(
      () => import("@/react/pages/settings/DatabasesPage"),
      (m) => m.DatabasesPage
    ),
  },
  {
    path: "environments",
    handle: { name: ENVIRONMENT_V1_ROUTE_DASHBOARD },
    lazy: lazyPage(
      () => import("@/react/pages/settings/EnvironmentsPage"),
      (m) => m.EnvironmentsPage
    ),
  },
  {
    path: "users/:principalEmail",
    handle: { name: WORKSPACE_ROUTE_USER_PROFILE },
    lazy: lazyPage(
      () => import("@/react/pages/settings/ProfilePage"),
      (m) => m.ProfilePage
    ),
  },
  {
    path: "403",
    handle: { name: WORKSPACE_ROUTE_403 },
    lazy: lazyPage(
      () => import("@/react/pages/workspace/Page403"),
      (m) => m.Page403
    ),
  },
  {
    path: "404",
    handle: { name: WORKSPACE_ROUTE_404 },
    lazy: lazyPage(
      () => import("@/react/pages/workspace/Page404"),
      (m) => m.Page404
    ),
  },
  // /sql-review — SettingRouteShell layout owns these leaves.
  {
    path: "sql-review",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        index: true,
        handle: { name: WORKSPACE_ROUTE_SQL_REVIEW },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SQLReviewPage"),
          (m) => m.SQLReviewPage
        ),
      },
      {
        path: "new",
        handle: { name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SQLReviewCreatePage"),
          (m) => m.SQLReviewCreatePage
        ),
      },
      {
        path: ":sqlReviewPolicySlug",
        handle: { name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SQLReviewDetailPage"),
          (m) => m.SQLReviewDetailPage
        ),
      },
    ],
  },
  // /idps — SettingRouteShell layout.
  {
    path: "idps",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        index: true,
        handle: { name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/IDPsPage"),
          (m) => m.IDPsPage
        ),
      },
      {
        path: ":idpId",
        handle: { name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/IDPDetailPage"),
          (m) => m.IDPDetailPage
        ),
      },
    ],
  },
  // Empty-path SettingRouteShell group (risk-assessment, members, roles, …).
  {
    path: "",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        path: "risk-assessment",
        handle: { name: WORKSPACE_ROUTE_RISK_ASSESSMENT },
        lazy: lazyPage(
          () => import("@/react/pages/settings/RiskAssessmentPage"),
          (m) => m.RiskAssessmentPage
        ),
      },
      {
        path: "custom-approval",
        handle: { name: WORKSPACE_ROUTE_CUSTOM_APPROVAL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/CustomApprovalPage"),
          (m) => m.CustomApprovalPage
        ),
      },
      {
        path: "global-masking",
        handle: { name: WORKSPACE_ROUTE_GLOBAL_MASKING },
        lazy: lazyPage(
          () => import("@/react/pages/settings/GlobalMaskingPage"),
          (m) => m.GlobalMaskingPage
        ),
      },
      {
        path: "semantic-types",
        handle: { name: WORKSPACE_ROUTE_SEMANTIC_TYPES },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SemanticTypesPage"),
          (m) => m.SemanticTypesPage
        ),
      },
      {
        path: "data-classification",
        handle: { name: WORKSPACE_ROUTE_DATA_CLASSIFICATION },
        lazy: lazyPage(
          () => import("@/react/pages/settings/DataClassificationPage"),
          (m) => m.DataClassificationPage
        ),
      },
      {
        path: "audit-log",
        handle: { name: WORKSPACE_ROUTE_AUDIT_LOG },
        lazy: lazyPage(
          () => import("@/react/pages/settings/AuditLogPage"),
          (m) => m.AuditLogPage
        ),
      },
      {
        path: "users",
        handle: { name: WORKSPACE_ROUTE_USERS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/UsersPage"),
          (m) => m.UsersPage
        ),
      },
      {
        path: "service-accounts",
        handle: { name: WORKSPACE_ROUTE_SERVICE_ACCOUNTS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/ServiceAccountsPage"),
          (m) => m.ServiceAccountsPage
        ),
      },
      {
        path: "workload-identities",
        handle: { name: WORKSPACE_ROUTE_WORKLOAD_IDENTITIES },
        lazy: lazyPage(
          () => import("@/react/pages/settings/WorkloadIdentitiesPage"),
          (m) => m.WorkloadIdentitiesPage
        ),
      },
      {
        path: "members",
        handle: { name: WORKSPACE_ROUTE_MEMBERS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/MembersPage"),
          (m) => m.MembersPage
        ),
      },
      {
        path: "groups",
        handle: { name: WORKSPACE_ROUTE_GROUPS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/GroupsPage"),
          (m) => m.GroupsPage
        ),
      },
      {
        path: "roles",
        handle: { name: WORKSPACE_ROUTE_ROLES },
        lazy: lazyPage(
          () => import("@/react/pages/settings/RolesPage"),
          (m) => m.RolesPage
        ),
      },
    ],
  },
  // /integration — SettingRouteShell layout.
  {
    path: "integration",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        path: "im",
        handle: { name: WORKSPACE_ROUTE_IM },
        lazy: lazyPage(
          () => import("@/react/pages/settings/IMPage"),
          (m) => m.IMPage
        ),
      },
      {
        path: "mcp",
        handle: { name: WORKSPACE_ROUTE_MCP },
        lazy: lazyPage(
          () => import("@/react/pages/settings/MCPPage"),
          (m) => m.MCPPage
        ),
      },
    ],
  },
];

// Workspace settings routes (`/setting/**`), SettingRouteShell layout.
const workspaceSettingRoutes: RouteObject[] = [
  {
    path: "setting",
    handle: { name: SETTING_ROUTE_WORKSPACE },
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        path: "profile",
        handle: { name: SETTING_ROUTE_PROFILE },
        lazy: lazyPage(
          () => import("@/react/pages/settings/ProfilePage"),
          (m) => m.ProfilePage
        ),
      },
      {
        path: "profile/two-factor",
        handle: { name: SETTING_ROUTE_PROFILE_TWO_FACTOR },
        lazy: lazyPage(
          () => import("@/react/pages/settings/TwoFactorSetupPage"),
          (m) => m.TwoFactorSetupPage
        ),
      },
      {
        path: "general",
        handle: { name: SETTING_ROUTE_WORKSPACE_GENERAL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/GeneralPage"),
          (m) => m.GeneralPage
        ),
      },
      {
        path: "subscription",
        handle: { name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SubscriptionPage"),
          (m) => m.SubscriptionPage
        ),
      },
    ],
  },
];

// Environment detail — redirect-only route in vue (it redirected to the
// environments dashboard with the environment name as a `#hash`). A leaf with
// no element/lazy renders a blank body, so the redirect is the element here.
const environmentV1Routes: RouteObject[] = [
  {
    path: "environments/:environmentName",
    handle: { name: "workspace.environment.detail" },
    loader: ({ params }) =>
      redirect(`/environments#${params.environmentName ?? ""}`),
  },
];

// Instance routes (`/instances/new`, `/instances/:instanceId/**`).
const instanceRoutes: RouteObject[] = [
  {
    path: "instances/new",
    handle: { name: INSTANCE_ROUTE_CREATE },
    lazy: lazyPage(
      () => import("@/react/pages/settings/CreateInstancePage"),
      (m) => m.CreateInstancePage
    ),
  },
  {
    path: "instances/:instanceId",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        index: true,
        handle: { name: INSTANCE_ROUTE_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/InstanceDetailPage"),
          (m) => m.InstanceDetailPage
        ),
      },
      {
        path: "databases/:databaseName",
        handle: { name: INSTANCE_ROUTE_DATABASE_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/settings/InstanceDatabaseRedirectPage"),
          (m) => m.InstanceDatabaseRedirectPage
        ),
      },
    ],
  },
];

// Project routes (`/projects/:projectId/**`), ProjectRouteShell layout.
const projectV1Routes: RouteObject[] = [
  {
    path: "projects/:projectId",
    element: <RouteShellOutletPlaceholder />,
    children: [
      {
        index: true,
        handle: { name: PROJECT_V1_ROUTE_DETAIL },
        // The project root has no page of its own — redirect to the Issues
        // tab (mirrors the legacy vue-router DETAIL → ISSUES redirect). `issues`
        // is relative to the parent `projects/:projectId`.
        element: <Navigate to="issues" replace />,
      },
      {
        path: "databases",
        handle: { name: PROJECT_V1_ROUTE_DATABASES },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectDatabasesPage"),
          (m) => m.ProjectDatabasesPage
        ),
      },
      {
        path: "access-grants",
        handle: { name: PROJECT_V1_ROUTE_ACCESS_GRANTS },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectAccessGrantsPage"),
          (m) => m.ProjectAccessGrantsPage
        ),
      },
      {
        path: "masking-exemption",
        children: [
          {
            index: true,
            handle: { name: PROJECT_V1_ROUTE_MASKING_EXEMPTION },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectMaskingExemptionPage"),
              (m) => m.ProjectMaskingExemptionPage
            ),
          },
          {
            path: "create",
            handle: { name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE },
            lazy: lazyPage(
              () =>
                import(
                  "@/react/pages/project/ProjectMaskingExemptionCreatePage"
                ),
              (m) => m.ProjectMaskingExemptionCreatePage
            ),
          },
        ],
      },
      {
        path: "database-groups",
        children: [
          {
            index: true,
            handle: { name: PROJECT_V1_ROUTE_DATABASE_GROUPS },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectDatabaseGroupsPage"),
              (m) => m.ProjectDatabaseGroupsPage
            ),
          },
          {
            path: "create",
            handle: { name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE },
            lazy: lazyPage(
              () =>
                import("@/react/pages/project/ProjectDatabaseGroupCreatePage"),
              (m) => m.ProjectDatabaseGroupCreatePage
            ),
          },
          {
            path: ":databaseGroupName",
            handle: { name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL },
            lazy: lazyPage(
              () =>
                import("@/react/pages/project/ProjectDatabaseGroupDetailPage"),
              (m) => m.ProjectDatabaseGroupDetailPage
            ),
          },
        ],
      },
      {
        path: "issues",
        handle: { name: PROJECT_V1_ROUTE_ISSUES },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectIssueDashboardPage"),
          (m) => m.ProjectIssueDashboardPage
        ),
      },
      {
        path: "plans",
        handle: { name: PROJECT_V1_ROUTE_PLANS },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectPlanDashboardPage"),
          (m) => m.ProjectPlanDashboardPage
        ),
      },
      {
        path: "sync-schema",
        handle: { name: PROJECT_V1_ROUTE_SYNC_SCHEMA },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectSyncSchemaPage"),
          (m) => m.ProjectSyncSchemaPage
        ),
      },
      {
        path: "audit-logs",
        handle: { name: PROJECT_V1_ROUTE_AUDIT_LOGS },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectAuditLogPage"),
          (m) => m.ProjectAuditLogPage
        ),
      },
      {
        path: "webhooks",
        children: [
          {
            index: true,
            handle: { name: PROJECT_V1_ROUTE_WEBHOOKS },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectWebhooksPage"),
              (m) => m.ProjectWebhooksPage
            ),
          },
          {
            path: "new",
            handle: { name: PROJECT_V1_ROUTE_WEBHOOK_CREATE },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectWebhookCreatePage"),
              (m) => m.ProjectWebhookCreatePage
            ),
          },
          {
            path: ":webhookResourceId",
            handle: { name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectWebhookDetailPage"),
              (m) => m.ProjectWebhookDetailPage
            ),
          },
        ],
      },
      {
        path: "members",
        handle: { name: PROJECT_V1_ROUTE_MEMBERS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/MembersPage"),
          (m) => m.MembersPage
        ),
      },
      {
        path: "service-accounts",
        handle: { name: PROJECT_V1_ROUTE_SERVICE_ACCOUNTS },
        lazy: lazyPage(
          () => import("@/react/pages/settings/ServiceAccountsPage"),
          (m) => m.ServiceAccountsPage
        ),
      },
      {
        path: "workload-identities",
        handle: { name: PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES },
        lazy: lazyPage(
          () => import("@/react/pages/settings/WorkloadIdentitiesPage"),
          (m) => m.WorkloadIdentitiesPage
        ),
      },
      {
        path: "settings",
        handle: { name: PROJECT_V1_ROUTE_SETTINGS },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectSettingsPage"),
          (m) => m.ProjectSettingsPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName",
        handle: { name: PROJECT_V1_ROUTE_DATABASE_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectDatabaseDetailPage"),
          (m) => m.ProjectDatabaseDetailPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName/changelogs/:changelogId",
        handle: { name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/project/DatabaseChangelogDetailPage"),
          (m) => m.DatabaseChangelogDetailPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName/revisions/:revisionId",
        handle: { name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/project/DatabaseRevisionDetailPage"),
          (m) => m.DatabaseRevisionDetailPage
        ),
      },
      {
        path: "data-export",
        handle: { name: PROJECT_V1_ROUTE_DATA_EXPORT },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectDataExportPage"),
          (m) => m.ProjectDataExportPage
        ),
      },
      {
        path: "releases",
        children: [
          {
            index: true,
            handle: { name: PROJECT_V1_ROUTE_RELEASES },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectReleaseDashboardPage"),
              (m) => m.ProjectReleaseDashboardPage
            ),
          },
          {
            path: ":releaseId",
            handle: { name: PROJECT_V1_ROUTE_RELEASE_DETAIL },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectReleaseDetailPage"),
              (m) => m.ProjectReleaseDetailPage
            ),
          },
        ],
      },
      {
        path: "gitops",
        handle: { name: PROJECT_V1_ROUTE_GITOPS },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectGitOpsPage"),
          (m) => m.ProjectGitOpsPage
        ),
      },
      // Plan detail — three routes share ProjectPlanDetailPage.
      {
        path: "plans/:planId",
        handle: { name: PROJECT_V1_ROUTE_PLAN_DETAIL },
        lazy: lazyPage(
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      {
        path: "plans/:planId/specs",
        handle: { name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS },
        lazy: lazyPage(
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      {
        path: "plans/:planId/specs/:specId",
        handle: { name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL },
        lazy: lazyPage(
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      // Issue detail.
      {
        path: "issues/:issueId",
        handle: { name: PROJECT_V1_ROUTE_ISSUE_DETAIL },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectIssueDetailPage"),
          (m) => m.ProjectIssueDetailPage
        ),
      },
      // Legacy rollout paths. The plan detail page now hosts the rollout: it
      // shows the deploy phase when the plan has a rollout and selects a
      // stage/task from the `?stageId=`/`?taskId=` query (see
      // usePlanDetailPage). So these path-based deep links redirect to the plan
      // detail page, converting the path stage/task into that query form so the
      // selection is preserved. The route names are kept so bookmarks resolve.
      {
        path: "plans/:planId/rollout",
        handle: { name: PROJECT_V1_ROUTE_PLAN_ROLLOUT },
        loader: ({ params }) =>
          redirect(`/projects/${params.projectId}/plans/${params.planId}`),
      },
      {
        path: "plans/:planId/rollout/stages/:stageId",
        handle: { name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE },
        loader: ({ params }) =>
          redirect(
            `/projects/${params.projectId}/plans/${params.planId}?stageId=${params.stageId}`
          ),
      },
      {
        path: "plans/:planId/rollout/stages/:stageId/tasks/:taskId",
        handle: { name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK },
        loader: ({ params }) =>
          redirect(
            `/projects/${params.projectId}/plans/${params.planId}?stageId=${params.stageId}&taskId=${params.taskId}`
          ),
      },
    ],
  },
];

// `/` DashboardLayout → BodyLayout child holding the dashboard routes, plus
// the `/issues` IssuesRouteShell route.
export const dashboardRoutes: RouteObject[] = [
  {
    path: "/",
    element: <DashboardLayout />,
    children: [
      {
        path: "",
        element: <BodyLayout />,
        children: [
          ...workspaceLevelRoutes,
          ...workspaceSettingRoutes,
          ...environmentV1Routes,
          ...instanceRoutes,
          ...projectV1Routes,
          // Workspace "My Issues" — lives under BodyLayout for the shared
          // dashboard header, but renders as a standalone full-width page
          // (header with logo, no sidebar): BodyLayout switches it to the
          // `issues` shell variant.
          {
            path: "issues",
            handle: { name: WORKSPACE_ROUTE_MY_ISSUES },
            lazy: lazyPage(
              () => import("@/react/pages/workspace/MyIssuesPage"),
              (m) => m.MyIssuesPage
            ),
          },
        ],
      },
    ],
  },
];
