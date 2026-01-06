import type { RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
import { t } from "@/plugins/i18n";
import { PROJECT_V1_ROUTE_DASHBOARD } from "./workspaceRoutes";

export const PROJECT_V1_ROUTE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.detail`;
export const PROJECT_V1_ROUTE_DATABASES = `${PROJECT_V1_ROUTE_DASHBOARD}.database`;
export const PROJECT_V1_ROUTE_MASKING_EXEMPTION = `${PROJECT_V1_ROUTE_DASHBOARD}.masking-exemption`;
export const PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.masking-exemption.create`;
export const PROJECT_V1_ROUTE_DATABASE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.database.detail`;
export const PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.database.changelog.detail`;
export const PROJECT_V1_ROUTE_DATABASE_GROUPS = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group`;
export const PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group.create`;
export const PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group.detail`;
export const PROJECT_V1_ROUTE_ISSUES = `${PROJECT_V1_ROUTE_DASHBOARD}.issue`;
export const PROJECT_V1_ROUTE_ISSUE_DETAIL = `${PROJECT_V1_ROUTE_ISSUES}.detail`;
export const PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 = `${PROJECT_V1_ROUTE_ISSUES}.detail.v1`;
export const PROJECT_V1_ROUTE_PLANS = `${PROJECT_V1_ROUTE_DASHBOARD}.plan`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.plan.detail`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS = `${PROJECT_V1_ROUTE_PLAN_DETAIL}.specs`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL = `${PROJECT_V1_ROUTE_PLAN_DETAIL}.spec.detail`;
export const PROJECT_V1_ROUTE_SYNC_SCHEMA = `${PROJECT_V1_ROUTE_DASHBOARD}.sync-schema`;
export const PROJECT_V1_ROUTE_AUDIT_LOGS = `${PROJECT_V1_ROUTE_DASHBOARD}.audit-logs`;
export const PROJECT_V1_ROUTE_WEBHOOKS = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook`;
export const PROJECT_V1_ROUTE_WEBHOOK_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.create`;
export const PROJECT_V1_ROUTE_WEBHOOK_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.detail`;
export const PROJECT_V1_ROUTE_MEMBERS = `${PROJECT_V1_ROUTE_DASHBOARD}.members`;
export const PROJECT_V1_ROUTE_SETTINGS = `${PROJECT_V1_ROUTE_DASHBOARD}.settings`;
export const PROJECT_V1_ROUTE_EXPORT_CENTER = `${PROJECT_V1_ROUTE_DASHBOARD}.export-center`;
export const PROJECT_V1_ROUTE_RELEASES = `${PROJECT_V1_ROUTE_DASHBOARD}.release`;
export const PROJECT_V1_ROUTE_RELEASE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.release.detail`;
export const PROJECT_V1_ROUTE_ROLLOUTS = `${PROJECT_V1_ROUTE_DASHBOARD}.rollouts`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT = `${PROJECT_V1_ROUTE_ROLLOUTS}.rollout`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE = `${PROJECT_V1_ROUTE_PLAN_ROLLOUT}.stage`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK = `${PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE}.task`;

const issueRoutes: RouteRecordRaw[] = [
  {
    path: "",
    component: () => import("@/views/project/IssueLayout.vue"),
    props: true,
    children: [
      {
        path: "plans/:planId",
        meta: {
          requiredPermissionList: () => [
            "bb.plans.get",
            "bb.planCheckRuns.get",
            "bb.taskRuns.list",
          ],
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_PLAN_DETAIL,
            redirect: {
              name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
            },
          },
          {
            path: "specs",
            name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
            component: () => import("@/views/project/PlanSpecsView.vue"),
          },
          {
            path: "specs/:specId",
            name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
            component: () => import("@/views/project/PlanSpecDetailView.vue"),
          },
        ],
      },
      {
        path: "issues/:issueId(\\d+)",
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
        meta: {
          requiredPermissionList: () => ["bb.issues.get"],
        },
        component: () => import("@/views/project/IssueDetailV1View.vue"),
        props: true,
      },
    ],
  },
];

// Rollout routes - nested under plans/:planId/rollout
const rolloutRoutes: RouteRecordRaw[] = [
  {
    path: "plans/:planId/rollout",
    component: () => import("@/views/project/RolloutLayout.vue"),
    props: true,
    meta: {
      title: () => t("common.rollout"),
    },
    children: [
      {
        path: "",
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
        component: () =>
          import("@/components/RolloutV1/components/RolloutView.vue"),
        props: true,
        meta: {
          requiredPermissionList: () => ["bb.rollouts.get", "bb.taskRuns.list"],
        },
      },
      {
        path: "stages/:stageId",
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
        component: () =>
          import("@/components/RolloutV1/components/RolloutView.vue"),
        props: true,
        meta: {
          requiredPermissionList: () => ["bb.rollouts.get", "bb.taskRuns.list"],
        },
      },
      {
        path: "stages/:stageId/tasks/:taskId",
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
        component: () =>
          import("@/components/RolloutV1/components/TaskView.vue"),
        props: true,
        meta: {
          requiredPermissionList: () => ["bb.rollouts.get", "bb.taskRuns.list"],
        },
      },
    ],
  },
];

const projectV1Routes: RouteRecordRaw[] = [
  {
    path: "projects/:projectId",
    components: {
      content: () => import("@/layouts/ProjectV1Layout.vue"),
      leftSidebar: ProjectSidebarV1,
    },
    props: { content: true, leftSidebar: true },
    meta: {
      requiredPermissionList: () => ["bb.projects.get"],
    },
    children: [
      {
        path: "",
        name: PROJECT_V1_ROUTE_DETAIL,
        // We will check user's permission to decide the redirect page.
        component: () => import("@/views/project/ProjectLandingPage.vue"),
        props: true,
      },
      {
        path: "databases",
        name: PROJECT_V1_ROUTE_DATABASES,
        meta: {
          title: () => t("common.databases"),
          requiredPermissionList: () => ["bb.databases.list"],
        },
        component: () => import("@/views/project/ProjectDatabaseDashboard.vue"),
        props: true,
      },
      {
        path: "masking-exemption",
        meta: {
          title: () => t("project.masking-exemption.self"),
          requiredPermissionList: () => ["bb.databases.get", "bb.policies.get"],
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
            component: () =>
              import("@/views/project/ProjectMaskingExemption.vue"),
            props: true,
          },
          {
            path: "create",
            name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
            component: () =>
              import("@/views/project/ProjectMaskingExemptionCreate.vue"),
            props: true,
            meta: {
              requiredPermissionList: () => [
                "bb.policies.create",
                "bb.databases.list",
              ],
            },
          },
        ],
      },
      {
        path: "database-groups",
        meta: {
          title: () => t("common.groups"),
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
            component: () =>
              import("@/views/project/ProjectDatabaseGroupDashboard.vue"),
            props: true,
            meta: {
              requiredPermissionList: () => ["bb.databaseGroups.list"],
            },
          },
          {
            path: "create",
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
            meta: {
              requiredPermissionList: () => [
                "bb.databaseGroups.create",
                "bb.databases.list",
              ],
            },
            component: () =>
              import("@/views/project/ProjectDatabaseGroupCreate.vue"),
            props: true,
          },
          {
            path: ":databaseGroupName",
            name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
            component: () =>
              import("@/views/project/ProjectDatabaseGroupDetail.vue"),
            props: true,
            meta: {
              requiredPermissionList: () => [
                "bb.databaseGroups.get",
                "bb.databases.list",
              ],
            },
          },
        ],
      },
      {
        path: "issues",
        meta: {
          title: () => t("common.issues"),
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_ISSUES,
            meta: {
              requiredPermissionList: () => ["bb.issues.list"],
            },
            component: () =>
              import("@/views/project/ProjectIssueDashboard.vue"),
            props: true,
          },
          {
            path: ":issueSlug",
            name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
            meta: {
              requiredPermissionList: () => ["bb.issues.get"],
            },
            component: () => import("@/views/project/ProjectIssueDetail.vue"),
            props: true,
          },
        ],
      },
      {
        path: "plans",
        name: PROJECT_V1_ROUTE_PLANS,
        meta: {
          title: () => t("plan.plans"),
          requiredPermissionList: () => ["bb.databases.list", "bb.plans.list"],
        },
        component: () => import("@/views/project/ProjectPlanDashboard.vue"),
        props: true,
      },
      {
        path: "rollouts",
        name: PROJECT_V1_ROUTE_ROLLOUTS,
        meta: {
          title: () => t("rollout.rollouts"),
          requiredPermissionList: () => ["bb.rollouts.list"],
        },
        component: () => import("@/views/project/ProjectRolloutDashboard.vue"),
        props: true,
      },
      {
        path: "sync-schema",
        name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
        meta: {
          title: () => t("database.sync-schema.title"),
          requiredPermissionList: () => [
            "bb.databases.sync",
            "bb.databases.list",
            "bb.databases.get",
            "bb.databases.getSchema",
            "bb.changelogs.get",
          ],
        },
        component: () =>
          import("@/views/project/ProjectSyncDatabasePanelV1.vue"),
        props: true,
      },
      {
        path: "audit-logs",
        name: PROJECT_V1_ROUTE_AUDIT_LOGS,
        meta: {
          title: () => t("settings.sidebar.audit-log"),
          requiredPermissionList: () => ["bb.auditLogs.search"],
        },
        component: () => import("@/views/project/ProjectAuditLogDashboard.vue"),
        props: true,
      },
      {
        path: "webhooks",
        meta: {
          title: () => t("common.webhooks"),
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_WEBHOOKS,
            component: () =>
              import("@/views/project/ProjectWebhookDashboard.vue"),
            props: true,
          },
          {
            path: "new",
            name: PROJECT_V1_ROUTE_WEBHOOK_CREATE,
            meta: {
              title: () => t("project.webhook.create-webhook"),
              requiredPermissionList: () => ["bb.projects.update"],
            },
            component: () => import("@/views/project/ProjectWebhookCreate.vue"),
            props: true,
          },
          {
            path: ":projectWebhookSlug",
            name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
            component: () => import("@/views/project/ProjectWebhookDetail.vue"),
            props: true,
          },
        ],
      },
      {
        path: "members",
        name: PROJECT_V1_ROUTE_MEMBERS,
        meta: {
          title: () => t("common.members"),
          requiredPermissionList: () => [
            "bb.projects.getIamPolicy",
            "bb.users.list",
            "bb.groups.list",
          ],
        },
        component: () => import("@/views/project/ProjectMemberDashboard.vue"),
        props: true,
      },
      {
        path: "settings",
        name: PROJECT_V1_ROUTE_SETTINGS,
        meta: {
          title: () => t("common.settings"),
          requiredPermissionList: () => ["bb.policies.list"],
        },
        component: () => import("@/views/project/ProjectSettingPanel.vue"),
        props: true,
      },
      {
        path: "instances/:instanceId/databases/:databaseName",
        meta: {
          title: () => t("common.database"),
          requiredPermissionList: () => ["bb.databases.get"],
        },
        component: () => import("@/views/project/ProjectDatabaseLayout.vue"),
        props: { content: true, leftSidebar: true },
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            component: () => import("@/views/DatabaseDetail"),
            props: true,
          },
          {
            path: "changelogs/:changelogId",
            name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
            meta: {
              requiredPermissionList: () => [
                "bb.databases.get",
                "bb.changelogs.get",
              ],
            },
            component: () =>
              import("@/views/DatabaseDetail/ChangelogDetail.vue"),
            props: (route) => ({
              ...route.params,
              project: `projects/${route.params.projectId}`,
              instance: `instances/${route.params.instanceId}`,
              database: `instances/${route.params.instanceId}/databases/${route.params.databaseName}`,
              changelogId: route.params.changelogId,
            }),
          },
        ],
      },
      {
        path: "export-center",
        name: PROJECT_V1_ROUTE_EXPORT_CENTER,
        meta: {
          title: () => t("export-center.self"),
          requiredPermissionList: () => ["bb.issues.list", "bb.databases.list"],
        },
        component: () => import("@/views/ExportCenter/index.vue"),
        props: true,
      },
      {
        path: "releases",
        meta: {
          title: () => t("release.releases"),
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_RELEASES,
            meta: {
              requiredPermissionList: () => ["bb.releases.list"],
            },
            component: () =>
              import("@/views/project/ProjectReleaseDashboard.vue"),
            props: true,
          },
          {
            path: ":releaseId",
            name: PROJECT_V1_ROUTE_RELEASE_DETAIL,
            meta: {
              requiredPermissionList: () => ["bb.releases.get"],
            },
            component: () => import("@/components/Release/ReleaseDetail/"),
            props: true,
          },
        ],
      },
      ...issueRoutes,
      ...rolloutRoutes,
    ],
  },
];

export default projectV1Routes;
