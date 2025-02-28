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
export const PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.database.revision.detail`;
export const PROJECT_V1_ROUTE_DATABASE_GROUPS = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group`;
export const PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group.create`;
export const PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.database-group.detail`;
export const PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG = `${PROJECT_V1_ROUTE_DASHBOARD}.deployment-config`;
export const PROJECT_V1_ROUTE_ISSUES = `${PROJECT_V1_ROUTE_DASHBOARD}.issue`;
export const PROJECT_V1_ROUTE_ISSUE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.issue.detail`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.plan.detail`;
export const PROJECT_V1_ROUTE_CHANGELISTS = `${PROJECT_V1_ROUTE_DASHBOARD}.changelist`;
export const PROJECT_V1_ROUTE_CHANGELIST_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.changelist.detail`;
export const PROJECT_V1_ROUTE_SYNC_SCHEMA = `${PROJECT_V1_ROUTE_DASHBOARD}.sync-schema`;
export const PROJECT_V1_ROUTE_SLOW_QUERIES = `${PROJECT_V1_ROUTE_DASHBOARD}.slow-queries`;
export const PROJECT_V1_ROUTE_ANOMALIES = `${PROJECT_V1_ROUTE_DASHBOARD}.anomalies`;
export const PROJECT_V1_ROUTE_AUDIT_LOGS = `${PROJECT_V1_ROUTE_DASHBOARD}.audit-logs`;
export const PROJECT_V1_ROUTE_WEBHOOKS = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook`;
export const PROJECT_V1_ROUTE_WEBHOOK_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.create`;
export const PROJECT_V1_ROUTE_WEBHOOK_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.detail`;
export const PROJECT_V1_ROUTE_MEMBERS = `${PROJECT_V1_ROUTE_DASHBOARD}.members`;
export const PROJECT_V1_ROUTE_SETTINGS = `${PROJECT_V1_ROUTE_DASHBOARD}.settings`;
export const PROJECT_V1_ROUTE_EXPORT_CENTER = `${PROJECT_V1_ROUTE_DASHBOARD}.export-center`;
export const PROJECT_V1_ROUTE_REVIEW_CENTER = `${PROJECT_V1_ROUTE_DASHBOARD}.review-center`;
export const PROJECT_V1_ROUTE_RELEASES = `${PROJECT_V1_ROUTE_DASHBOARD}.release`;
export const PROJECT_V1_ROUTE_RELEASE_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.release.create`;
export const PROJECT_V1_ROUTE_RELEASE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.release.detail`;
export const PROJECT_V1_ROUTE_ROLLOUTS = `${PROJECT_V1_ROUTE_DASHBOARD}.rollout`;
export const PROJECT_V1_ROUTE_ROLLOUT_DETAIL = `${PROJECT_V1_ROUTE_ROLLOUTS}.detail`;
export const PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL = `${PROJECT_V1_ROUTE_ROLLOUT_DETAIL}.task.detail`;

const projectV1Routes: RouteRecordRaw[] = [
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
        // We will check user's permission to decide the redirect page.
        component: () => import("@/views/project/ProjectLandingPage.vue"),
        props: true,
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
        path: "masking-exemption",
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.list",
            "bb.policies.get",
          ],
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
              requiredProjectPermissionList: () => ["bb.policies.create"],
            },
          },
        ],
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
            path: "create",
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS_CREATE,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => ["bb.projects.update"],
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
        path: "issues/:issueSlug",
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () => import("@/views/project/ProjectIssueDetail.vue"),
        props: true,
      },
      {
        path: "plans/:planSlug",
        name: PROJECT_V1_ROUTE_PLAN_DETAIL,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () => import("@/views/project/ProjectPlanDetail.vue"),
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
        component: () =>
          import("@/views/project/ProjectSyncDatabasePanelV1.vue"),
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
          requiredProjectPermissionList: () => ["bb.projects.get"],
        },
        component: () =>
          import("@/views/project/ProjectAnomalyCenterDashboard.vue"),
        props: true,
      },
      {
        path: "audit-logs",
        name: PROJECT_V1_ROUTE_AUDIT_LOGS,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.auditLogs.search",
          ],
        },
        component: () => import("@/views/project/ProjectAuditLogDashboard.vue"),
        props: true,
      },
      {
        path: "webhooks",
        meta: {
          overrideTitle: true,
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
      {
        path: "instances/:instanceId/databases/:databaseName",
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.databases.get",
          ],
        },
        component: () => import("@/views/project/ProjectDatabaseLayout.vue"),
        props: { content: true, leftSidebar: true },
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            meta: {
              overrideTitle: true,
            },
            component: () => import("@/views/DatabaseDetail"),
            props: true,
          },
          {
            path: "changelogs/:changelogId",
            name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
            meta: {
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.databases.get",
                // TODO: add changelogs related permission.
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
          {
            path: "revisions/:revisionId",
            name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.databases.get",
              ],
            },
            component: () =>
              import("@/views/DatabaseDetail/RevisionDetail.vue"),
            props: (route) => ({
              ...route.params,
              project: `projects/${route.params.projectId}`,
              instance: `instances/${route.params.instanceId}`,
              database: `instances/${route.params.instanceId}/databases/${route.params.databaseName}`,
              revision: `instances/${route.params.instanceId}/databases/${route.params.databaseName}/revisions/${route.params.revisionId}`,
            }),
          },
        ],
      },
      {
        path: "export-center",
        name: PROJECT_V1_ROUTE_EXPORT_CENTER,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.issues.list",
            "bb.databases.list",
          ],
        },
        component: () => import("@/views/ExportCenter/index.vue"),
        props: true,
      },
      {
        path: "review-center",
        name: PROJECT_V1_ROUTE_REVIEW_CENTER,
        meta: {
          overrideTitle: true,
          requiredProjectPermissionList: () => [
            "bb.projects.get",
            "bb.issues.list",
            "bb.databases.list",
          ],
        },
        component: () => import("@/views/ReviewCenter/index.vue"),
        props: true,
      },
      {
        path: "releases",
        meta: {
          overrideTitle: true,
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_RELEASES,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.releases.list",
              ],
            },
            component: () =>
              import("@/views/project/ProjectReleaseDashboard.vue"),
            props: true,
          },
          {
            path: "new",
            name: PROJECT_V1_ROUTE_RELEASE_CREATE,
            meta: {
              title: () => t("release.create"),
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.releases.create",
              ],
            },
            component: () => import("@/components/Release/ReleaseCreate/"),
            props: true,
          },
          {
            path: ":releaseId",
            name: PROJECT_V1_ROUTE_RELEASE_DETAIL,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => [
                "bb.projects.get",
                "bb.releases.get",
              ],
            },
            component: () => import("@/components/Release/ReleaseDetail/"),
            props: true,
          },
        ],
      },
      {
        path: "rollouts",
        meta: {
          overrideTitle: true,
        },
        props: true,
        children: [
          {
            path: "",
            name: PROJECT_V1_ROUTE_ROLLOUTS,
            meta: {
              overrideTitle: true,
              requiredProjectPermissionList: () => ["bb.projects.get"],
            },
            component: () =>
              import("@/views/project/ProjectRolloutDashboard.vue"),
            props: true,
          },
          {
            path: ":rolloutId",
            component: () =>
              import(
                "@/components/Rollout/RolloutDetail/RolloutDetailLayout.vue"
              ),
            props: true,
            children: [
              {
                path: "",
                name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
                meta: {
                  overrideTitle: true,
                  requiredProjectPermissionList: () => ["bb.projects.get"],
                },
                component: () => import("@/components/Rollout/RolloutDetail/"),
                props: true,
              },
              {
                path: "stages/:stageId/tasks/:taskId",
                name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
                meta: {
                  overrideTitle: true,
                },
                component: () =>
                  import("@/components/Rollout/RolloutDetail/TaskDetail/"),
                props: true,
              },
            ],
          },
        ],
      },
    ],
  },
];

export default projectV1Routes;
