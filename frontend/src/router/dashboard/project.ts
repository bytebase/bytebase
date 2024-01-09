import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import ProjectSidebar from "@/components/Project/ProjectSidebar.vue";
import { t } from "@/plugins/i18n";
import { useProjectV1Store, useProjectWebhookV1Store } from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { idFromSlug } from "@/utils";
import DashboardSidebar from "@/views/DashboardSidebar.vue";

const projectRoutes: RouteRecordRaw[] = [
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
    path: "project/:projectSlug",
    components: {
      content: () => import("@/layouts/ProjectLayout.vue"),
      leftSidebar: ProjectSidebar,
    },
    props: { content: true, leftSidebar: true },
    children: [
      {
        path: "",
        name: "workspace.project.detail",
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.projectSlug as string;
            const projectId = idFromSlug(slug);
            if (projectId === DEFAULT_PROJECT_ID) {
              return t("database.unassigned-databases");
            }
            const projectV1 = useProjectV1Store().getProjectByUID(
              String(projectId)
            );
            return projectV1.title;
          },
        },
        component: () => import("@/views/ProjectDetail.vue"),
        props: true,
      },
      {
        path: "webhook/new",
        name: "workspace.project.hook.create",
        meta: {
          title: () => t("project.webhook.create-webhook"),
        },
        component: () => import("@/views/ProjectWebhookCreate.vue"),
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
            const project = useProjectV1Store().getProjectByUID(
              String(idFromSlug(projectSlug))
            );
            const webhook =
              useProjectWebhookV1Store().getProjectWebhookFromProjectById(
                project,
                idFromSlug(projectWebhookSlug)
              );

            return `${t("common.webhook")} - ${webhook?.title ?? "unknown"}`;
          },
        },
        component: () => import("@/views/ProjectWebhookDetail.vue"),
        props: true,
      },
      {
        path: "branches/:branchName",
        name: "workspace.project.branch.detail",
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/views/branch/BranchDetail.vue"),
        props: true,
      },
      {
        path: "branches/:branchName/rollout",
        name: "workspace.project.branch.rollout",
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/views/branch/BranchRollout.vue"),
        props: true,
      },
      {
        path: "branches/:branchName/merge",
        name: "workspace.project.branch.merge",
        meta: {
          title: () => t("branch.merge-rebase.merge-branch"),
        },
        component: () => import("@/views/branch/BranchMerge.vue"),
        props: true,
      },
      {
        path: "branches/:branchName/rebase",
        name: "workspace.project.branch.rebase",
        meta: {
          title: () => t("branch.merge-rebase.rebase-branch"),
        },
        component: () => import("@/views/branch/BranchRebase.vue"),
        props: true,
      },
      {
        path: "changelists/:changelistName",
        name: "workspace.project.changelist.detail",
        meta: {
          overrideTitle: true,
        },
        component: () => import("@/components/Changelist/ChangelistDetail/"),
        props: true,
      },
      {
        path: "database-groups/:databaseGroupName",
        name: "workspace.project.database-group.detail",
        component: () => import("@/views/DatabaseGroupDetail.vue"),
        props: true,
      },
      {
        path: "database-groups/:databaseGroupName/table-groups/:schemaGroupName",
        name: "workspace.project.database-group.table-group.detail",
        component: () => import("@/views/SchemaGroupDetail.vue"),
        props: true,
      },
    ],
  },
];

export default projectRoutes;
