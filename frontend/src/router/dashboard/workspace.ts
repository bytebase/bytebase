import type { RouteRecordRaw } from "vue-router";
import DummyRootView from "@/DummyRootView";
import { t } from "@/plugins/i18n";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
  WORKSPACE_ROUTE_AUDIT_LOG,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MCP,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_RISK_CENTER,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_SEMANTIC_TYPES,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_USERS,
} from "./workspaceRoutes";

const rootRoute: RouteRecordRaw = {
  path: "",
  name: WORKSPACE_ROOT_MODULE,
  components: {
    content: DummyRootView,
  },
};

const workspaceRoutes: RouteRecordRaw[] = [
  rootRoute,
  {
    path: "landing",
    name: WORKSPACE_ROUTE_LANDING,
    components: {
      content: () => import("@/views/DashboardLanding.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
  },
  {
    path: "projects",
    name: PROJECT_V1_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.projects"),
    },
    components: {
      content: () => import("@/views/ProjectDashboard.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "instances",
    name: INSTANCE_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.instances"),
      requiredPermissionList: () => ["bb.instances.list"],
    },
    components: {
      content: () => import("@/views/InstanceDashboard.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "databases",
    name: DATABASE_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.databases"),
      requiredPermissionList: () => ["bb.databases.list"],
    },
    components: {
      content: () => import("@/views/DatabaseDashboard.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "environments",
    name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
    meta: {
      title: () => t("common.environments"),
      requiredPermissionList: () => ["bb.settings.get", "bb.policies.get"],
    },
    components: {
      content: () => import("@/views/EnvironmentDashboard.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "users/:principalEmail",
    name: WORKSPACE_ROUTE_USER_PROFILE,
    components: {
      content: () => import("@/views/ProfileDashboard.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: true,
  },
  {
    path: "403",
    name: WORKSPACE_ROUTE_403,
    components: {
      content: () => import("@/views/Page403.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "404",
    name: WORKSPACE_ROUTE_404,
    components: {
      content: () => import("@/views/Page404.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: {
      content: true,
      leftSidebar: true,
    },
  },
  {
    path: "sql-review",
    meta: {
      title: () => t("sql-review.title"),
      requiredPermissionList: () => ["bb.policies.get"],
    },
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: {
      content: true,
      leftSidebar: true,
    },
    children: [
      {
        path: "",
        name: WORKSPACE_ROUTE_SQL_REVIEW,
        meta: {
          title: () => t("sql-review.title"),
          requiredPermissionList: () => ["bb.reviewConfigs.list"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReview.vue"),
        props: true,
      },
      {
        path: "new",
        name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
        meta: {
          title: () => t("sql-review.create.breadcrumb"),
          requiredPermissionList: () => ["bb.reviewConfigs.create"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewCreate.vue"),
        props: true,
      },
      {
        path: ":sqlReviewPolicySlug",
        name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
        meta: {
          title: () => t("sql-review.title"),
          requiredPermissionList: () => ["bb.reviewConfigs.get"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewDetail.vue"),
        props: true,
      },
    ],
  },
  {
    path: "idps",
    meta: {
      title: () => t("settings.sidebar.sso"),
      requiredPermissionList: () => ["bb.identityProviders.get"],
    },
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: {
      content: true,
      leftSidebar: true,
    },
    children: [
      {
        path: "",
        name: WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
        component: () => import("@/views/SettingWorkspaceSSO.vue"),
      },
      {
        path: ":idpId",
        name: WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
        props: true,
      },
    ],
  },
  {
    path: "",
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: () => import("@/views/DashboardSidebar.vue"),
    },
    props: true,
    children: [
      {
        path: "risk-center",
        name: WORKSPACE_ROUTE_RISK_CENTER,
        meta: {
          title: () => t("custom-approval.risk.self"),
        },
        component: () => import("@/views/SettingWorkspaceRiskCenter.vue"),
        props: true,
      },
      {
        path: "custom-approval",
        name: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
        meta: {
          title: () => t("custom-approval.self"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceCustomApproval.vue"),
        props: true,
      },
      {
        path: "global-masking",
        name: WORKSPACE_ROUTE_GLOBAL_MASKING,
        meta: {
          title: () => t("settings.sidebar.global-masking"),
          requiredPermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceDataMasking.vue"),
        props: true,
      },
      {
        path: "semantic-types",
        name: WORKSPACE_ROUTE_SEMANTIC_TYPES,
        meta: {
          title: () => t("settings.sensitive-data.semantic-types.self"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceSemanticTypes.vue"),
        props: true,
      },
      {
        path: "data-classification",
        name: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
        meta: {
          title: () => t("settings.sidebar.data-classification"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () =>
          import("@/views/SettingWorkspaceDataClassification.vue"),
        props: true,
      },
      {
        path: "audit-log",
        name: WORKSPACE_ROUTE_AUDIT_LOG,
        meta: {
          title: () => t("settings.sidebar.audit-log"),
          requiredPermissionList: () => ["bb.auditLogs.search"],
        },
        component: () => import("@/views/SettingWorkspaceAuditLog.vue"),
        props: true,
      },
      {
        path: "users",
        name: WORKSPACE_ROUTE_USERS,
        meta: {
          title: () => t("settings.sidebar.users-and-groups"),
          requiredPermissionList: () => ["bb.users.list", "bb.groups.list"],
        },
        component: () => import("@/views/SettingWorkspaceUsers.vue"),
        props: true,
      },
      {
        path: "members",
        name: WORKSPACE_ROUTE_MEMBERS,
        meta: {
          title: () => t("settings.sidebar.members"),
          requiredPermissionList: () => [
            "bb.workspaces.getIamPolicy",
            "bb.users.list",
            "bb.groups.list",
          ],
        },
        component: () => import("@/views/SettingWorkspaceMembers.vue"),
        props: true,
      },
      {
        path: "roles",
        name: WORKSPACE_ROUTE_ROLES,
        meta: {
          title: () => t("settings.sidebar.custom-roles"),
          requiredPermissionList: () => ["bb.roles.list"],
        },
        component: () => import("@/views/SettingWorkspaceRole.vue"),
        props: true,
      },
      {
        path: "im",
        name: WORKSPACE_ROUTE_IM,
        meta: {
          title: () => t("settings.sidebar.im-integration"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceIM.vue"),
        props: true,
      },
      {
        path: "mcp",
        name: WORKSPACE_ROUTE_MCP,
        meta: {
          title: () => t("settings.sidebar.mcp"),
          requiredPermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceMCP.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceRoutes;
