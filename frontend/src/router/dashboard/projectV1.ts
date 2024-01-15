import { RouteRecordRaw } from "vue-router";
import ProjectSidebarV1 from "@/components/Project/ProjectSidebarV1.vue";
import { t } from "@/plugins/i18n";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

export const PROJECT_V1_ROUTE = "workspace.projectV1";
export const PROJECT_V1_ROUTE_DETAIL = `${PROJECT_V1_ROUTE}.detail`;
export const PROJECT_V1_ROUTE_DATABASES = `${PROJECT_V1_ROUTE}.database.dashboard`;
export const PROJECT_V1_ROUTE_DATABASE_GROUPS = `${PROJECT_V1_ROUTE}.database-group.dashboard`;
export const PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL = `${PROJECT_V1_ROUTE}.database-group.detail`;
export const PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL = `${PROJECT_V1_ROUTE}.database-group.table-group.detail`;
export const PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG = `${PROJECT_V1_ROUTE}.deployment-config`;
export const PROJECT_V1_ROUTE_BRANCHES = `${PROJECT_V1_ROUTE}.branch.dashboard`;
export const PROJECT_V1_ROUTE_BRANCH_DETAIL = `${PROJECT_V1_ROUTE}.branch.detail`;
export const PROJECT_V1_ROUTE_BRANCH_ROLLOUT = `${PROJECT_V1_ROUTE}.branch.rollout`;
export const PROJECT_V1_ROUTE_BRANCH_MERGE = `${PROJECT_V1_ROUTE}.branch.merge`;
export const PROJECT_V1_ROUTE_BRANCH_REBASE = `${PROJECT_V1_ROUTE}.branch.rebase`;
export const PROJECT_V1_ROUTE_ISSUES = `${PROJECT_V1_ROUTE}.issue.dashboard`;
export const PROJECT_V1_ROUTE_CHANGE_HISTORIES = `${PROJECT_V1_ROUTE}.change-histories.dashboard`;
export const PROJECT_V1_ROUTE_CHANGELISTS = `${PROJECT_V1_ROUTE}.changelist.dashboard`;
export const PROJECT_V1_ROUTE_CHANGELIST_DETAIL = `${PROJECT_V1_ROUTE}.changelist.detail`;
export const PROJECT_V1_ROUTE_SYNC_SCHEMA = `${PROJECT_V1_ROUTE}.sync-schema`;
export const PROJECT_V1_ROUTE_SLOW_QUERIES = `${PROJECT_V1_ROUTE}.slow-queries`;
export const PROJECT_V1_ROUTE_ANOMALIES = `${PROJECT_V1_ROUTE}.anomalies`;
export const PROJECT_V1_ROUTE_ACTIVITIES = `${PROJECT_V1_ROUTE}.activities`;
export const PROJECT_V1_ROUTE_GITOPS = `${PROJECT_V1_ROUTE}.gitops`;
export const PROJECT_V1_ROUTE_WEBHOOKS = `${PROJECT_V1_ROUTE}.webhook.dashboard`;
export const PROJECT_V1_ROUTE_WEBHOOK_CREATE = `${PROJECT_V1_ROUTE}.webhook.create`;
export const PROJECT_V1_ROUTE_WEBHOOK_DETAIL = `${PROJECT_V1_ROUTE}.webhook.detail`;
export const PROJECT_V1_ROUTE_MEMBERS = `${PROJECT_V1_ROUTE}.members`;
export const PROJECT_V1_ROUTE_SETTINGS = `${PROJECT_V1_ROUTE}.settings`;

const projectV1Routes: RouteRecordRaw[] = [
  {
    path: "project",
    name: "workspace.project",
    meta: {
      title: () => t("common.projects"),
      getQuickActionList: () => {
        return ["quickaction.bb.project.create"];
      },
    },
    components: {
      content: () => import("@/views/ProjectDashboard.vue"),
      leftSidebar: DashboardSidebar,
    },
    props: { content: true, leftSidebar: true },
  },
  {
    path: "projects/:projectId",
    components: {
      content: () => import("@/layouts/ProjectV1Layout.vue"),
      leftSidebar: ProjectSidebarV1,
    },
    props: { content: true, leftSidebar: true },
    children: [
      {
        path: "",
        name: PROJECT_V1_ROUTE_DETAIL,
        redirect: { name: PROJECT_V1_ROUTE_DATABASES },
      },
      {
        path: "databases",
        name: PROJECT_V1_ROUTE_DATABASES,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.list",
          ],
        },
        component: () => import("@/views/project/ProjectDatabaseDashboard.vue"),
        props: true,
      },
      {
        path: "database-groups",
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
            meta: {
              overrideTitle: true,
            },
            component: () =>
              import("@/views/project/ProjectDatabaseGroupDashboard.vue"),
            props: true,
          },
          {
            path: ":databaseGroupName",
            name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
            component: () => import("@/views/DatabaseGroupDetail.vue"),
            props: true,
          },
          {
            path: ":databaseGroupName/table-groups/:schemaGroupName",
            name: PROJECT_V1_ROUTE_DATABASE_GROUP_TABLE_GROUP_DETAIL,
            component: () => import("@/views/SchemaGroupDetail.vue"),
            props: true,
          },
        ],
      },
      {
        path: "deployment-config",
        name: PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () =>
          import("@/views/project/ProjectDeploymentConfigPanel.vue"),
        props: true,
      },
      {
        path: "branches",
        meta: {
          overrideTitle: true,
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_BRANCHES,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.branches.list",
              ],
            },
            component: () =>
              import("@/views/project/ProjectBrancheDashboard.vue"),
            props: true,
          },
          {
            path: ":branchName",
            name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.branches.get",
              ],
            },
            component: () => import("@/views/branch/BranchDetail.vue"),
            props: true,
          },
          {
            path: ":branchName/rollout",
            name: PROJECT_V1_ROUTE_BRANCH_ROLLOUT,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.branches.update",
              ],
            },
            component: () => import("@/views/branch/BranchRollout.vue"),
            props: true,
          },
          {
            path: ":branchName/merge",
            name: PROJECT_V1_ROUTE_BRANCH_MERGE,
            meta: {
              title: () => t("branch.merge-rebase.merge-branch"),
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.branches.update",
              ],
            },
            component: () => import("@/views/branch/BranchMerge.vue"),
            props: true,
          },
          {
            path: ":branchName/rebase",
            name: PROJECT_V1_ROUTE_BRANCH_REBASE,
            meta: {
              title: () => t("branch.merge-rebase.rebase-branch"),
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.branches.update",
              ],
            },
            component: () => import("@/views/branch/BranchRebase.vue"),
            props: true,
          },
        ],
      },
      {
        path: "issues",
        name: PROJECT_V1_ROUTE_ISSUES,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.issues.list",
          ],
        },
        component: () => import("@/views/project/ProjectIssueDashboard.vue"),
        props: true,
      },
      {
        path: "change-histories",
        name: PROJECT_V1_ROUTE_CHANGE_HISTORIES,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.changeHistories.list",
          ],
        },
        component: () =>
          import("@/views/project/ProjectChangeHistoryDashboard.vue"),
        props: true,
      },
      {
        path: "changelists",
        meta: {
          overrideTitle: true,
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_CHANGELISTS,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.changelists.list",
              ],
            },
            component: () =>
              import("@/views/project/ProjectChangelistDashboard.vue"),
            props: true,
          },
          {
            path: ":changelistName",
            name: PROJECT_V1_ROUTE_CHANGELIST_DETAIL,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.changelists.get",
              ],
            },
            component: () =>
              import("@/components/Changelist/ChangelistDetail/"),
            props: true,
          },
        ],
      },
      {
        path: "sync-schema",
        name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.sync",
          ],
        },
        component: () => import("@/views/project/ProjectSyncDatabasePanel.vue"),
        props: true,
      },
      {
        path: "slow-queries",
        name: PROJECT_V1_ROUTE_SLOW_QUERIES,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.slowQueries.list",
          ],
        },
        component: () =>
          import("@/views/project/ProjectSlowQueryDashboard.vue"),
        props: true,
      },
      {
        path: "anomalies",
        name: PROJECT_V1_ROUTE_ANOMALIES,
        meta: {
          overrideTitle: true,
          // TODO(ed): permission check
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () =>
          import("@/views/project/ProjectAnomalyCenterDashboard.vue"),
        props: true,
      },
      {
        path: "activities",
        name: PROJECT_V1_ROUTE_ACTIVITIES,
        meta: {
          overrideTitle: true,
          // TODO(ed): permission check
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () => import("@/views/project/ProjectActivityDashboard.vue"),
        props: true,
      },
      {
        path: "gitops",
        name: PROJECT_V1_ROUTE_GITOPS,
        meta: {
          overrideTitle: true,
          // TODO(ed): permission check
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () =>
          import("@/views/project/ProjectVersionControlPanel.vue"),
        props: true,
      },
      {
        path: "webhooks",
        meta: {
          overrideTitle: true,
          // TODO(ed): permission check
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_WEBHOOKS,
            meta: {
              overrideTitle: true,
            },
            component: () =>
              import("@/views/project/ProjectWebhookDashboard.vue"),
            props: true,
          },
          {
            path: "new",
            name: PROJECT_V1_ROUTE_WEBHOOK_CREATE,
            meta: {
              title: () => t("project.webhook.create-webhook"),
              requiredProjectPermissionList: () => ["bb.projects.update"],
            },
            component: () => import("@/views/project/ProjectWebhookCreate.vue"),
            props: true,
          },
          {
            path: ":projectWebhookSlug",
            name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
            meta: {
              overrideTitle: true,
            },
            component: () => import("@/views/project/ProjectWebhookDetail.vue"),
            props: true,
          },
        ],
      },
      {
        path: "members",
        name: PROJECT_V1_ROUTE_MEMBERS,
        meta: {
          overrideTitle: true,
          // TODO(ed): permission check
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.projects.getIamPolicy",
          ],
        },
        component: () => import("@/views/project/ProjectMemberDashboard.vue"),
        props: true,
      },
      {
        path: "settings",
        name: PROJECT_V1_ROUTE_SETTINGS,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () => import("@/views/project/ProjectSettingPanel.vue"),
        props: true,
      },
    ],
  },
];

export default projectV1Routes;
