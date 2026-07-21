import { Navigate, type RouteObject, redirect } from "react-router";
import { BodyLayout } from "@/app/layouts/BodyLayout";
import { DashboardLayout } from "@/app/layouts/DashboardLayout";
import { RouteErrorPage } from "@/app/RouteErrorPage";
import { rootGuard } from "@/app/router/guard";
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
} from "@/app/router/handles";
import { issueDetailRedirectLoader } from "@/app/router/issueDetailRedirect";
import { lazyPage } from "@/app/router/lazyPage";
import { ProjectRouteGate } from "@/app/router/ProjectRouteGate";
import { RouteGroupOutlet } from "@/app/router/RouteGroupOutlet";
import type { Permission } from "@/types";

// Workspace and project routes nested under the dashboard layouts. Leaf route
// modules are lazy-loaded from their owner under src/routes.

// Workspace-level pages (the BodyLayout children).
const workspaceLevelRoutes: RouteObject[] = [
  {
    index: true,
    // The bare workspace root ("/") has no page; it redirects dynamically
    // (change-mode → SQL Editor, else last visit / landing). The global guard
    // on the root route only re-runs on initial document load, NOT on
    // client-side navigations (the root route is already matched, so its loader
    // is not revalidated) — so after an in-app `navigate("/")` (e.g. post-login)
    // "/" would render this element-less index and show a blank page until a
    // manual refresh. This index-route loader runs `rootGuard` whenever "/" is
    // freshly matched (client nav AND initial load, since the index route is
    // newly matched each time), so the redirect always fires. `guardRedirect`
    // records the no-element redirect for the route-reachability test.
    handle: { name: WORKSPACE_ROOT_MODULE, guardRedirect: true },
    loader: ({ request }) =>
      rootGuard({
        name: WORKSPACE_ROOT_MODULE,
        url: new URL(request.url),
      }),
  },
  {
    path: "landing",
    handle: { name: WORKSPACE_ROUTE_LANDING },
    lazy: lazyPage(
      () => import("@/routes/workspace/LandingPage"),
      (m) => m.LandingPage
    ),
  },
  {
    path: "projects",
    handle: { name: PROJECT_V1_ROUTE_DASHBOARD },
    lazy: lazyPage(
      () => import("@/routes/workspace/ProjectsPage"),
      (m) => m.ProjectsPage
    ),
  },
  {
    path: "instances",
    handle: {
      name: INSTANCE_ROUTE_DASHBOARD,
      requiredPermissionList: (): Permission[] => ["bb.instances.list"],
    },
    lazy: lazyPage(
      () => import("@/routes/workspace/InstancesPage"),
      (m) => m.InstancesPage
    ),
  },
  {
    path: "databases",
    handle: {
      name: DATABASE_ROUTE_DASHBOARD,
      requiredPermissionList: (): Permission[] => ["bb.databases.list"],
    },
    lazy: lazyPage(
      () => import("@/routes/workspace/DatabasesPage"),
      (m) => m.DatabasesPage
    ),
  },
  {
    path: "environments",
    handle: {
      name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      requiredPermissionList: (): Permission[] => [
        "bb.settings.getEnvironment",
      ],
    },
    lazy: lazyPage(
      () => import("@/routes/workspace/EnvironmentsPage"),
      (m) => m.EnvironmentsPage
    ),
  },
  {
    path: "users/:principalEmail",
    handle: { name: WORKSPACE_ROUTE_USER_PROFILE },
    lazy: lazyPage(
      () => import("@/routes/workspace/ProfilePage"),
      (m) => m.ProfilePage
    ),
  },
  {
    path: "403",
    handle: { name: WORKSPACE_ROUTE_403 },
    lazy: lazyPage(
      () => import("@/routes/workspace/Page403"),
      (m) => m.Page403
    ),
  },
  {
    path: "404",
    handle: { name: WORKSPACE_ROUTE_404 },
    lazy: lazyPage(
      () => import("@/routes/workspace/Page404"),
      (m) => m.Page404
    ),
  },
  // /sql-review — SettingRouteShell layout owns these leaves.
  {
    path: "sql-review",
    element: <RouteGroupOutlet />,
    children: [
      {
        index: true,
        handle: {
          name: WORKSPACE_ROUTE_SQL_REVIEW,
          requiredPermissionList: (): Permission[] => [
            "bb.reviewConfigs.list",
            "bb.policies.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/SQLReviewPage"),
          (m) => m.SQLReviewPage
        ),
      },
      {
        path: "new",
        handle: {
          name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
          requiredPermissionList: (): Permission[] => [
            "bb.reviewConfigs.create",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/SQLReviewCreatePage"),
          (m) => m.SQLReviewCreatePage
        ),
      },
      {
        path: ":sqlReviewPolicySlug",
        handle: {
          name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
          requiredPermissionList: (): Permission[] => [
            "bb.reviewConfigs.get",
            "bb.policies.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/SQLReviewDetailPage"),
          (m) => m.SQLReviewDetailPage
        ),
      },
    ],
  },
  // /idps — SettingRouteShell layout. The parent carries the route permission
  // so both children (list + detail) inherit it via the matched-chain
  // aggregation in `assembleRoute`.
  {
    path: "idps",
    handle: {
      requiredPermissionList: (): Permission[] => ["bb.identityProviders.get"],
    },
    element: <RouteGroupOutlet />,
    children: [
      {
        index: true,
        handle: { name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS },
        lazy: lazyPage(
          () => import("@/routes/workspace/IDPsPage"),
          (m) => m.IDPsPage
        ),
      },
      {
        path: ":idpId",
        handle: { name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL },
        lazy: lazyPage(
          () => import("@/routes/workspace/IDPDetailPage"),
          (m) => m.IDPDetailPage
        ),
      },
    ],
  },
  // Empty-path SettingRouteShell group (risk-assessment, members, roles, …).
  {
    path: "",
    element: <RouteGroupOutlet />,
    children: [
      {
        path: "risk-assessment",
        handle: { name: WORKSPACE_ROUTE_RISK_ASSESSMENT },
        lazy: lazyPage(
          () => import("@/routes/workspace/RiskAssessmentPage"),
          (m) => m.RiskAssessmentPage
        ),
      },
      {
        path: "custom-approval",
        handle: {
          name: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/CustomApprovalPage"),
          (m) => m.CustomApprovalPage
        ),
      },
      {
        path: "global-masking",
        handle: {
          name: WORKSPACE_ROUTE_GLOBAL_MASKING,
          requiredPermissionList: (): Permission[] => [
            "bb.policies.getMaskingRulePolicy",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/GlobalMaskingPage"),
          (m) => m.GlobalMaskingPage
        ),
      },
      {
        path: "semantic-types",
        handle: {
          name: WORKSPACE_ROUTE_SEMANTIC_TYPES,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/SemanticTypesPage"),
          (m) => m.SemanticTypesPage
        ),
      },
      {
        path: "data-classification",
        handle: {
          name: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/DataClassificationPage"),
          (m) => m.DataClassificationPage
        ),
      },
      {
        path: "audit-log",
        handle: {
          name: WORKSPACE_ROUTE_AUDIT_LOG,
          requiredPermissionList: (): Permission[] => ["bb.auditLogs.search"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/AuditLogPage"),
          (m) => m.AuditLogPage
        ),
      },
      {
        path: "users",
        handle: {
          name: WORKSPACE_ROUTE_USERS,
          requiredPermissionList: (): Permission[] => ["bb.users.list"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/UsersPage"),
          (m) => m.UsersPage
        ),
      },
      {
        path: "service-accounts",
        handle: {
          name: WORKSPACE_ROUTE_SERVICE_ACCOUNTS,
          requiredPermissionList: (): Permission[] => [
            "bb.serviceAccounts.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/ServiceAccountsPage"),
          (m) => m.ServiceAccountsPage
        ),
      },
      {
        path: "workload-identities",
        handle: {
          name: WORKSPACE_ROUTE_WORKLOAD_IDENTITIES,
          requiredPermissionList: (): Permission[] => [
            "bb.workloadIdentities.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/WorkloadIdentitiesPage"),
          (m) => m.WorkloadIdentitiesPage
        ),
      },
      {
        path: "members",
        handle: {
          name: WORKSPACE_ROUTE_MEMBERS,
          requiredPermissionList: (): Permission[] => [
            "bb.workspaces.getIamPolicy",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/MembersPage"),
          (m) => m.MembersPage
        ),
      },
      {
        path: "groups",
        handle: {
          name: WORKSPACE_ROUTE_GROUPS,
          requiredPermissionList: (): Permission[] => ["bb.groups.list"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/GroupsPage"),
          (m) => m.GroupsPage
        ),
      },
      {
        path: "roles",
        handle: {
          name: WORKSPACE_ROUTE_ROLES,
          requiredPermissionList: (): Permission[] => ["bb.roles.list"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/RolesPage"),
          (m) => m.RolesPage
        ),
      },
    ],
  },
  // /integration — SettingRouteShell layout.
  {
    path: "integration",
    element: <RouteGroupOutlet />,
    children: [
      {
        path: "im",
        handle: {
          name: WORKSPACE_ROUTE_IM,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/IMPage"),
          (m) => m.IMPage
        ),
      },
      {
        path: "mcp",
        handle: {
          name: WORKSPACE_ROUTE_MCP,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/MCPPage"),
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
    element: <RouteGroupOutlet />,
    children: [
      {
        path: "profile",
        handle: { name: SETTING_ROUTE_PROFILE },
        lazy: lazyPage(
          () => import("@/routes/workspace/ProfilePage"),
          (m) => m.ProfilePage
        ),
      },
      {
        path: "profile/two-factor",
        handle: { name: SETTING_ROUTE_PROFILE_TWO_FACTOR },
        lazy: lazyPage(
          () => import("@/routes/workspace/TwoFactorSetupPage"),
          (m) => m.TwoFactorSetupPage
        ),
      },
      {
        path: "general",
        handle: {
          name: SETTING_ROUTE_WORKSPACE_GENERAL,
          requiredPermissionList: (): Permission[] => [
            "bb.settings.getWorkspaceProfile",
            "bb.policies.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/GeneralPage"),
          (m) => m.GeneralPage
        ),
      },
      {
        path: "subscription",
        handle: {
          name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/SubscriptionPage"),
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
    handle: {
      name: "workspace.environment.detail",
      requiredPermissionList: (): Permission[] => [
        "bb.settings.getEnvironment",
      ],
    },
    loader: ({ params }) =>
      redirect(`/environments#${params.environmentName ?? ""}`),
  },
];

// Instance routes (`/instances/new`, `/instances/:instanceId/**`).
const instanceRoutes: RouteObject[] = [
  {
    path: "instances/new",
    handle: {
      name: INSTANCE_ROUTE_CREATE,
      requiredPermissionList: (): Permission[] => ["bb.instances.create"],
    },
    lazy: lazyPage(
      () => import("@/routes/workspace/CreateInstancePage"),
      (m) => m.CreateInstancePage
    ),
  },
  {
    path: "instances/:instanceId",
    element: <RouteGroupOutlet />,
    children: [
      {
        index: true,
        handle: {
          name: INSTANCE_ROUTE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.instances.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/InstanceDetailPage"),
          (m) => m.InstanceDetailPage
        ),
      },
      {
        path: "databases/:databaseName",
        handle: {
          name: INSTANCE_ROUTE_DATABASE_DETAIL,
          requiredPermissionList: (): Permission[] => [
            "bb.projects.get",
            "bb.databases.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/InstanceDatabaseRedirectPage"),
          (m) => m.InstanceDatabaseRedirectPage
        ),
      },
    ],
  },
];

// Project routes (`/projects/:projectId/**`).
//
// The `requiredPermissionList` entries on the parent (`bb.projects.get`) and
// each leaf are ported 1:1 from the legacy vue routes and aggregate via
// `assembleRoute` into `route.requiredPermissions`. `ProjectRouteGate` (the
// parent element) loads the project and enforces those permissions before its
// `<Outlet/>` mounts the leaf — project-scoped checks need the loaded `Project`
// resource, which is why `BodyLayout` routes project routes straight to this
// gate instead of its generic workspace-level `RoutePermissionGuardShell`.
const projectV1Routes: RouteObject[] = [
  {
    path: "projects/:projectId",
    handle: {
      requiredPermissionList: (): Permission[] => ["bb.projects.get"],
    },
    element: <ProjectRouteGate />,
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
        handle: {
          name: PROJECT_V1_ROUTE_DATABASES,
          requiredPermissionList: (): Permission[] => ["bb.databases.list"],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectDatabasesPage"),
          (m) => m.ProjectDatabasesPage
        ),
      },
      {
        path: "access-grants",
        handle: { name: PROJECT_V1_ROUTE_ACCESS_GRANTS },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectAccessGrantsPage"),
          (m) => m.ProjectAccessGrantsPage
        ),
      },
      {
        path: "masking-exemption",
        children: [
          {
            index: true,
            handle: {
              name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
              requiredPermissionList: (): Permission[] => [
                "bb.databases.get",
                "bb.policies.getMaskingExemptionPolicy",
              ],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectMaskingExemptionPage"),
              (m) => m.ProjectMaskingExemptionPage
            ),
          },
          {
            path: "create",
            handle: {
              name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
              requiredPermissionList: (): Permission[] => [
                "bb.policies.createMaskingExemptionPolicy",
                "bb.policies.updateMaskingExemptionPolicy",
                "bb.databases.list",
                "bb.databaseCatalogs.get",
              ],
            },
            lazy: lazyPage(
              () =>
                import("@/routes/project/ProjectMaskingExemptionCreatePage"),
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
            handle: {
              name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
              requiredPermissionList: (): Permission[] => [
                "bb.databaseGroups.list",
              ],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectDatabaseGroupsPage"),
              (m) => m.ProjectDatabaseGroupsPage
            ),
          },
          {
            path: "create",
            handle: {
              name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
              requiredPermissionList: (): Permission[] => [
                "bb.databaseGroups.create",
                "bb.databases.list",
              ],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectDatabaseGroupCreatePage"),
              (m) => m.ProjectDatabaseGroupCreatePage
            ),
          },
          {
            path: ":databaseGroupName",
            handle: {
              name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
              requiredPermissionList: (): Permission[] => [
                "bb.databaseGroups.get",
                "bb.databases.list",
              ],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectDatabaseGroupDetailPage"),
              (m) => m.ProjectDatabaseGroupDetailPage
            ),
          },
        ],
      },
      {
        path: "issues",
        handle: {
          name: PROJECT_V1_ROUTE_ISSUES,
          requiredPermissionList: (): Permission[] => ["bb.issues.list"],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectIssueDashboardPage"),
          (m) => m.ProjectIssueDashboardPage
        ),
      },
      {
        path: "plans",
        handle: {
          name: PROJECT_V1_ROUTE_PLANS,
          requiredPermissionList: (): Permission[] => [
            "bb.databases.list",
            "bb.plans.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectPlanDashboardPage"),
          (m) => m.ProjectPlanDashboardPage
        ),
      },
      {
        path: "sync-schema",
        handle: {
          name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
          requiredPermissionList: (): Permission[] => [
            "bb.databases.sync",
            "bb.databases.list",
            "bb.databases.get",
            "bb.databases.getSchema",
            "bb.changelogs.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectSyncSchemaPage"),
          (m) => m.ProjectSyncSchemaPage
        ),
      },
      {
        path: "audit-logs",
        handle: {
          name: PROJECT_V1_ROUTE_AUDIT_LOGS,
          requiredPermissionList: (): Permission[] => ["bb.auditLogs.search"],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectAuditLogPage"),
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
              () => import("@/routes/project/ProjectWebhooksPage"),
              (m) => m.ProjectWebhooksPage
            ),
          },
          {
            path: "new",
            handle: {
              name: PROJECT_V1_ROUTE_WEBHOOK_CREATE,
              requiredPermissionList: (): Permission[] => [
                "bb.projects.update",
              ],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectWebhookCreatePage"),
              (m) => m.ProjectWebhookCreatePage
            ),
          },
          {
            path: ":webhookResourceId",
            handle: { name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectWebhookDetailPage"),
              (m) => m.ProjectWebhookDetailPage
            ),
          },
        ],
      },
      {
        path: "members",
        handle: {
          name: PROJECT_V1_ROUTE_MEMBERS,
          requiredPermissionList: (): Permission[] => [
            "bb.projects.getIamPolicy",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/MembersPage"),
          (m) => m.MembersPage
        ),
      },
      {
        path: "service-accounts",
        handle: {
          name: PROJECT_V1_ROUTE_SERVICE_ACCOUNTS,
          requiredPermissionList: (): Permission[] => [
            "bb.serviceAccounts.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/ServiceAccountsPage"),
          (m) => m.ServiceAccountsPage
        ),
      },
      {
        path: "workload-identities",
        handle: {
          name: PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES,
          requiredPermissionList: (): Permission[] => [
            "bb.workloadIdentities.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/workspace/WorkloadIdentitiesPage"),
          (m) => m.WorkloadIdentitiesPage
        ),
      },
      {
        path: "settings",
        handle: { name: PROJECT_V1_ROUTE_SETTINGS },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectSettingsPage"),
          (m) => m.ProjectSettingsPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName",
        handle: {
          name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.databases.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectDatabaseDetailPage"),
          (m) => m.ProjectDatabaseDetailPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName/changelogs/:changelogId",
        handle: {
          name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
          requiredPermissionList: (): Permission[] => [
            "bb.databases.get",
            "bb.changelogs.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/DatabaseChangelogDetailPage"),
          (m) => m.DatabaseChangelogDetailPage
        ),
      },
      {
        path: "instances/:instanceId/databases/:databaseName/revisions/:revisionId",
        handle: {
          name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.databases.get"],
        },
        lazy: lazyPage(
          () => import("@/routes/project/DatabaseRevisionDetailPage"),
          (m) => m.DatabaseRevisionDetailPage
        ),
      },
      {
        path: "releases",
        children: [
          {
            index: true,
            handle: {
              name: PROJECT_V1_ROUTE_RELEASES,
              requiredPermissionList: (): Permission[] => ["bb.releases.list"],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectReleaseDashboardPage"),
              (m) => m.ProjectReleaseDashboardPage
            ),
          },
          {
            path: ":releaseId",
            handle: {
              name: PROJECT_V1_ROUTE_RELEASE_DETAIL,
              requiredPermissionList: (): Permission[] => ["bb.releases.get"],
            },
            lazy: lazyPage(
              () => import("@/routes/project/ProjectReleaseDetailPage"),
              (m) => m.ProjectReleaseDetailPage
            ),
          },
        ],
      },
      {
        path: "gitops",
        handle: {
          name: PROJECT_V1_ROUTE_GITOPS,
          requiredPermissionList: (): Permission[] => [
            "bb.workloadIdentities.list",
            "bb.databases.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/ProjectGitOpsPage"),
          (m) => m.ProjectGitOpsPage
        ),
      },
      // Plan detail — three routes share ProjectPlanDetailPage.
      {
        path: "plans/:planId",
        handle: {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          requiredPermissionList: (): Permission[] => [
            "bb.plans.get",
            "bb.planCheckRuns.get",
            "bb.taskRuns.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      {
        path: "plans/:planId/specs",
        handle: {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          requiredPermissionList: (): Permission[] => [
            "bb.plans.get",
            "bb.planCheckRuns.get",
            "bb.taskRuns.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      {
        path: "plans/:planId/specs/:specId",
        handle: {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          requiredPermissionList: (): Permission[] => [
            "bb.plans.get",
            "bb.planCheckRuns.get",
            "bb.taskRuns.list",
          ],
        },
        lazy: lazyPage(
          () => import("@/routes/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      // Issue detail. Schema/data change issues redirect to Plan Detail — the
      // canonical review surface (BYT-9721); create-database, export, and grant
      // issues stay here. The `loader` (static) decides the redirect and the
      // `lazy` Component renders when it doesn't.
      {
        path: "issues/:issueId",
        handle: {
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.issues.get"],
        },
        loader: issueDetailRedirectLoader,
        lazy: lazyPage(
          () => import("@/routes/project/ProjectIssueDetailPage"),
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
          {
            // Layout-seam error boundary (pathless): a crashing page
            // renders the recovery panel inside BodyLayout's content
            // area, keeping the sidebar + header alive. Placing the
            // errorElement on the BodyLayout route itself would replace
            // the whole shell.
            errorElement: <RouteErrorPage inline />,
            children: [
              ...workspaceLevelRoutes,
              ...workspaceSettingRoutes,
              ...environmentV1Routes,
              ...instanceRoutes,
              ...projectV1Routes,
              // Workspace "My Issues" — lives under BodyLayout for the
              // shared dashboard header, but renders as a standalone
              // full-width page (header with logo, no sidebar):
              // BodyLayout switches it to the `issues` shell variant.
              {
                path: "issues",
                handle: { name: WORKSPACE_ROUTE_MY_ISSUES },
                lazy: lazyPage(
                  () => import("@/routes/workspace/MyIssuesPage"),
                  (m) => m.MyIssuesPage
                ),
              },
            ],
          },
        ],
      },
    ],
  },
];
