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
export const PROJECT_V1_ROUTE_ISSUE_DETAIL = `${PROJECT_V1_ROUTE}.issue.detail`;
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
      quickActionListByRole: () => {
        return new Map([
          ["OWNER", ["quickaction.bb.project.create"]],
          ["DBA", ["quickaction.bb.project.create"]],
          ["DEVELOPER", ["quickaction.bb.project.create"]],
        ]);
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
        },
        component: () => import("@/views/project/ProjectDatabaseDashboard.vue"),
        props: true,
      },
      {
        path: "database-groups",
        meta: {
          overrideTitle: true,
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
            },
            component: () =>
              import("@/views/project/ProjectBranchDashboard.vue"),
            props: true,
          },
          {
            path: ":branchName",
            name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
            meta: {
              overrideTitle: true,
            },
            component: () => import("@/views/branch/BranchDetail.vue"),
            props: true,
          },
          {
            path: ":branchName/rollout",
            name: PROJECT_V1_ROUTE_BRANCH_ROLLOUT,
            meta: {
              overrideTitle: true,
            },
            component: () => import("@/views/branch/BranchRollout.vue"),
            props: true,
          },
          {
            path: ":branchName/merge",
            name: PROJECT_V1_ROUTE_BRANCH_MERGE,
            meta: {
              title: () => t("branch.merge-rebase.merge-branch"),
            },
            component: () => import("@/views/branch/BranchMerge.vue"),
            props: true,
          },
          {
            path: ":branchName/rebase",
            name: PROJECT_V1_ROUTE_BRANCH_REBASE,
            meta: {
              title: () => t("branch.merge-rebase.rebase-branch"),
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
        },
        component: () => import("@/views/project/ProjectIssueDashboard.vue"),
        props: true,
      },
      {
        path: "issues/:issueSlug",
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/views/project/ProjectIssueDetail.vue"),
        props: true,
      },
    ],
  },
];

export default projectV1Routes;
