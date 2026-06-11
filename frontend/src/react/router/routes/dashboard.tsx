import { Navigate, type RouteObject, redirect } from "react-router-dom";
import { BodyLayout } from "@/react/app/layouts/BodyLayout";
import { DashboardLayout } from "@/react/app/layouts/DashboardLayout";
import { RouteErrorPage } from "@/react/app/RouteErrorPage";
import { rootGuard } from "@/react/router/guard";
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
import { ProjectRouteGate } from "@/react/router/ProjectRouteGate";
import type { Permission } from "@/types";

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
    handle: {
      name: INSTANCE_ROUTE_DASHBOARD,
      requiredPermissionList: (): Permission[] => ["bb.instances.list"],
    },
    lazy: lazyPage(
      () => import("@/react/pages/settings/InstancesPage"),
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
      () => import("@/react/pages/settings/DatabasesPage"),
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
        handle: {
          name: WORKSPACE_ROUTE_SQL_REVIEW,
          requiredPermissionList: (): Permission[] => [
            "bb.reviewConfigs.list",
            "bb.policies.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/SQLReviewPage"),
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
          () => import("@/react/pages/settings/SQLReviewCreatePage"),
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
          () => import("@/react/pages/settings/SQLReviewDetailPage"),
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
        handle: {
          name: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/CustomApprovalPage"),
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
          () => import("@/react/pages/settings/GlobalMaskingPage"),
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
          () => import("@/react/pages/settings/SemanticTypesPage"),
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
          () => import("@/react/pages/settings/DataClassificationPage"),
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
          () => import("@/react/pages/settings/AuditLogPage"),
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
          () => import("@/react/pages/settings/UsersPage"),
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
          () => import("@/react/pages/settings/ServiceAccountsPage"),
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
          () => import("@/react/pages/settings/WorkloadIdentitiesPage"),
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
          () => import("@/react/pages/settings/MembersPage"),
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
          () => import("@/react/pages/settings/GroupsPage"),
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
        handle: {
          name: WORKSPACE_ROUTE_IM,
          requiredPermissionList: (): Permission[] => ["bb.settings.get"],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/IMPage"),
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
        handle: {
          name: SETTING_ROUTE_WORKSPACE_GENERAL,
          requiredPermissionList: (): Permission[] => [
            "bb.settings.getWorkspaceProfile",
            "bb.policies.get",
          ],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/GeneralPage"),
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
        handle: {
          name: INSTANCE_ROUTE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.instances.get"],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/InstanceDetailPage"),
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
          () => import("@/react/pages/settings/InstanceDatabaseRedirectPage"),
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
            handle: {
              name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
              requiredPermissionList: (): Permission[] => [
                "bb.databases.get",
                "bb.policies.getMaskingExemptionPolicy",
              ],
            },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectMaskingExemptionPage"),
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
            handle: {
              name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
              requiredPermissionList: (): Permission[] => [
                "bb.databaseGroups.list",
              ],
            },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectDatabaseGroupsPage"),
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
              () =>
                import("@/react/pages/project/ProjectDatabaseGroupCreatePage"),
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
              () =>
                import("@/react/pages/project/ProjectDatabaseGroupDetailPage"),
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
          () => import("@/react/pages/project/ProjectIssueDashboardPage"),
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
          () => import("@/react/pages/project/ProjectPlanDashboardPage"),
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
          () => import("@/react/pages/project/ProjectSyncSchemaPage"),
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
            handle: {
              name: PROJECT_V1_ROUTE_WEBHOOK_CREATE,
              requiredPermissionList: (): Permission[] => [
                "bb.projects.update",
              ],
            },
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
        handle: {
          name: PROJECT_V1_ROUTE_MEMBERS,
          requiredPermissionList: (): Permission[] => [
            "bb.projects.getIamPolicy",
          ],
        },
        lazy: lazyPage(
          () => import("@/react/pages/settings/MembersPage"),
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
          () => import("@/react/pages/settings/ServiceAccountsPage"),
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
        handle: {
          name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.databases.get"],
        },
        lazy: lazyPage(
          () => import("@/react/pages/project/ProjectDatabaseDetailPage"),
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
          () => import("@/react/pages/project/DatabaseChangelogDetailPage"),
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
          () => import("@/react/pages/project/DatabaseRevisionDetailPage"),
          (m) => m.DatabaseRevisionDetailPage
        ),
      },
      {
        path: "data-export",
        handle: {
          name: PROJECT_V1_ROUTE_DATA_EXPORT,
          requiredPermissionList: (): Permission[] => [
            "bb.issues.list",
            "bb.databases.list",
          ],
        },
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
            handle: {
              name: PROJECT_V1_ROUTE_RELEASES,
              requiredPermissionList: (): Permission[] => ["bb.releases.list"],
            },
            lazy: lazyPage(
              () => import("@/react/pages/project/ProjectReleaseDashboardPage"),
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
              () => import("@/react/pages/project/ProjectReleaseDetailPage"),
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
          () => import("@/react/pages/project/ProjectGitOpsPage"),
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
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
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
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
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
          () =>
            import("@/react/pages/project/plan-detail/ProjectPlanDetailPage"),
          (m) => m.ProjectPlanDetailPage
        ),
      },
      // Issue detail.
      {
        path: "issues/:issueId",
        handle: {
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          requiredPermissionList: (): Permission[] => ["bb.issues.get"],
        },
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
                  () => import("@/react/pages/workspace/MyIssuesPage"),
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
