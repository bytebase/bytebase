import { startCase } from "lodash-es";
import type { RouteRecordRaw } from "vue-router";
import DummyRootView from "@/DummyRootView";
import { t } from "@/plugins/i18n";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROOT_MODULE,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_USER_PROFILE,
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
  WORKSPACE_ROUTE_SCHEMA_TEMPLATE,
  WORKSPACE_ROUTE_CUSTOM_APPROVAL,
  WORKSPACE_ROUTE_RISK_CENTER,
  WORKSPACE_ROUTE_DATA_MASKING,
  WORKSPACE_ROUTE_DATA_CLASSIFICATION,
  WORKSPACE_ROUTE_AUDIT_LOG,
  WORKSPACE_ROUTE_GITOPS,
  WORKSPACE_ROUTE_GITOPS_CREATE,
  WORKSPACE_ROUTE_GITOPS_DETAIL,
  WORKSPACE_ROUTE_SSO,
  WORKSPACE_ROUTE_SSO_CREATE,
  WORKSPACE_ROUTE_SSO_DETAIL,
  WORKSPACE_ROUTE_MAIL_DELIVERY,
  WORKSPACE_ROUTE_USERS,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_ROLES,
  WORKSPACE_ROUTE_IM,
  DATABASE_ROUTE_DASHBOARD,
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
      requiredWorkspacePermissionList: () => ["bb.instances.list"],
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
      requiredWorkspacePermissionList: () => ["bb.databases.list"],
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
      requiredWorkspacePermissionList: () => ["bb.environments.list"],
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
    name: "error.403",
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
    name: "error.404",
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
      requiredWorkspacePermissionList: () => ["bb.policies.get"],
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
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReview.vue"),
        props: true,
      },
      {
        path: "new",
        name: WORKSPACE_ROUTE_SQL_REVIEW_CREATE,
        meta: {
          title: () => t("sql-review.create.breadcrumb"),
          requiredWorkspacePermissionList: () => ["bb.policies.create"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewCreate.vue"),
        props: true,
      },
      {
        path: ":sqlReviewPolicySlug",
        name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
        meta: {
          title: () => t("sql-review.title"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewDetail.vue"),
        props: true,
      },
    ],
  },
  {
    path: "gitops",
    meta: {
      title: () => t("settings.sidebar.gitops"),
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
        name: WORKSPACE_ROUTE_GITOPS,
        meta: {
          requiredWorkspacePermissionList: () => ["bb.vcsProviders.list"],
        },
        component: () => import("@/views/SettingWorkspaceVCS.vue"),
        props: true,
      },
      {
        path: "new",
        name: WORKSPACE_ROUTE_GITOPS_CREATE,
        meta: {
          title: () => t("repository.add-git-provider"),
          requiredWorkspacePermissionList: () => ["bb.vcsProviders.create"],
        },
        component: () => import("@/views/SettingWorkspaceVCSCreate.vue"),
        props: true,
      },
      {
        path: ":vcsResourceId",
        name: WORKSPACE_ROUTE_GITOPS_DETAIL,
        meta: {
          requiredWorkspacePermissionList: () => ["bb.vcsProviders.get"],
        },
        component: () => import("@/views/SettingWorkspaceVCSDetail.vue"),
        props: true,
      },
    ],
  },
  {
    path: "sso",
    meta: {
      title: () => t("settings.sidebar.sso"),
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
        name: WORKSPACE_ROUTE_SSO,
        meta: {
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceSSO.vue"),
      },
      {
        path: "new",
        name: WORKSPACE_ROUTE_SSO_CREATE,
        meta: {
          requiredWorkspacePermissionList: () => [
            "bb.identityProviders.create",
          ],
        },
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
      },
      {
        path: ":ssoId",
        name: WORKSPACE_ROUTE_SSO_DETAIL,
        meta: {
          requiredWorkspacePermissionList: () => ["bb.identityProviders.get"],
        },
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
        path: "custom-approval",
        name: WORKSPACE_ROUTE_CUSTOM_APPROVAL,
        meta: {
          title: () => t("custom-approval.self"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceCustomApproval.vue"),
        props: true,
      },
      {
        path: "schema-template",
        name: WORKSPACE_ROUTE_SCHEMA_TEMPLATE,
        meta: {
          title: () => startCase(t("schema-template.self")),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSchemaTemplate.vue"),
        props: true,
      },
      {
        path: "risk-center",
        name: WORKSPACE_ROUTE_RISK_CENTER,
        meta: {
          title: () => t("custom-approval.risk.risk-center"),
          requiredWorkspacePermissionList: () => [
            "bb.settings.get",
            "bb.risks.list",
          ],
        },
        component: () => import("@/views/SettingWorkspaceRiskCenter.vue"),
        props: true,
      },
      {
        path: "data-masking",
        name: WORKSPACE_ROUTE_DATA_MASKING,
        meta: {
          title: () => t("settings.sidebar.data-masking"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceDataMasking.vue"),
        props: true,
      },
      {
        path: "data-classification",
        name: WORKSPACE_ROUTE_DATA_CLASSIFICATION,
        meta: {
          title: () => t("settings.sidebar.data-classification"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
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
          requiredWorkspacePermissionList: () => [
            "bb.settings.get",
            "bb.auditLogs.search",
          ],
        },
        component: () => import("@/views/SettingWorkspaceAuditLog.vue"),
        props: true,
      },
      {
        path: "mail-delivery",
        name: WORKSPACE_ROUTE_MAIL_DELIVERY,
        meta: {
          title: () => t("settings.sidebar.mail-delivery"),
          requiredWorkspacePermissionList: () => [
            "bb.settings.get",
            "bb.settings.set",
          ],
        },
        component: () => import("@/views/SettingWorkspaceMailDelivery.vue"),
      },
      {
        path: "users",
        name: WORKSPACE_ROUTE_USERS,
        meta: {
          title: () => t("settings.sidebar.users-and-groups"),
        },
        component: () => import("@/views/SettingWorkspaceUsers.vue"),
        props: true,
      },
      {
        path: "members",
        name: WORKSPACE_ROUTE_MEMBERS,
        meta: {
          title: () => t("settings.sidebar.members"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceMembers.vue"),
        props: true,
      },
      {
        path: "roles",
        name: WORKSPACE_ROUTE_ROLES,
        meta: {
          title: () => t("settings.sidebar.custom-roles"),
          requiredWorkspacePermissionList: () => ["bb.roles.list"],
        },
        component: () => import("@/views/SettingWorkspaceRole.vue"),
        props: true,
      },
      {
        path: "im",
        name: WORKSPACE_ROUTE_IM,
        meta: {
          title: () => t("settings.sidebar.im-integration"),
          requiredWorkspacePermissionList: () => [
            "bb.settings.get",
            "bb.settings.set",
          ],
        },
        component: () => import("@/views/SettingWorkspaceIM.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceRoutes;
