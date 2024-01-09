import { startCase } from "lodash-es";
import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import {
  useIdentityProviderStore,
  useSQLReviewStore,
  useVCSV1Store,
} from "@/store";
import { idFromSlug } from "@/utils";
import SettingSidebar from "@/views/SettingSidebar.vue";

const workspaceSettingRoutes: RouteRecordRaw[] = [
  {
    path: "setting",
    name: "setting",
    meta: { title: () => t("common.settings") },
    components: {
      content: () => import("@/layouts/SettingLayout.vue"),
      leftSidebar: SettingSidebar,
    },
    props: {
      content: true,
      leftSidebar: true,
    },
    children: [
      {
        path: "",
        name: "setting.profile",
        meta: { title: () => t("settings.sidebar.profile") },
        component: () => import("@/views/ProfileDashboard.vue"),
        alias: "profile",
        props: true,
      },
      {
        path: "profile/two-factor",
        name: "setting.profile.two-factor",
        meta: {
          title: () => t("two-factor.self"),
        },
        component: () => import("@/views/TwoFactorSetup.vue"),
        props: true,
      },
      {
        path: "general",
        name: "setting.workspace.general",
        meta: { title: () => t("settings.sidebar.general") },
        component: () => import("@/views/SettingWorkspaceGeneral.vue"),
        props: true,
      },
      {
        path: "agent",
        name: "setting.workspace.agent",
        meta: { title: () => t("common.agents") },
        component: () => import("@/views/SettingWorkspaceAgent.vue"),
        props: true,
      },
      {
        path: "member",
        name: "setting.workspace.member",
        meta: { title: () => t("settings.sidebar.members") },
        component: () => import("@/views/SettingWorkspaceMember.vue"),
        props: true,
      },
      {
        path: "role",
        name: "setting.workspace.role",
        meta: { title: () => t("settings.sidebar.custom-roles") },
        component: () => import("@/views/SettingWorkspaceRole.vue"),
        props: true,
      },
      {
        path: "sso",
        name: "setting.workspace.sso",
        meta: { title: () => t("settings.sidebar.sso") },
        component: () => import("@/views/SettingWorkspaceSSO.vue"),
      },
      {
        path: "sso/new",
        name: "setting.workspace.sso.create",
        meta: { title: () => t("settings.sidebar.sso") },
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
      },
      {
        path: "sso/:ssoName",
        name: "setting.workspace.sso.detail",
        meta: {
          title: (route: RouteLocationNormalized) => {
            const name = route.params.ssoName as string;
            return (
              useIdentityProviderStore().getIdentityProviderByName(name)
                ?.title || t("settings.sidebar.sso")
            );
          },
        },
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
        props: true,
      },
      {
        path: "sensitive-data",
        name: "setting.workspace.sensitive-data",
        meta: { title: () => t("settings.sidebar.sensitive-data") },
        component: () => import("@/views/SettingWorkspaceSensitiveData.vue"),
        props: true,
      },
      {
        path: "access-control",
        name: "setting.workspace.access-control",
        meta: { title: () => t("settings.sidebar.access-control") },
        component: () => import("@/views/SettingWorkspaceAccessControl.vue"),
        props: true,
      },
      {
        path: "risk-center",
        name: "setting.workspace.risk-center",
        meta: { title: () => t("custom-approval.risk.risk-center") },
        component: () => import("@/views/SettingWorkspaceRiskCenter.vue"),
        props: true,
      },
      {
        path: "custom-approval",
        name: "setting.workspace.custom-approval",
        meta: { title: () => t("custom-approval.self") },
        component: () => import("@/views/SettingWorkspaceCustomApproval.vue"),
        props: true,
      },
      {
        path: "slow-query",
        name: "setting.workspace.slow-query",
        meta: { title: () => startCase(t("slow-query.self")) },
        component: () => import("@/views/SettingWorkspaceSlowQuery.vue"),
        props: true,
      },
      {
        path: "schema-template",
        name: "setting.workspace.schema-template",
        meta: { title: () => startCase(t("schema-template.self")) },
        component: () => import("@/views/SettingWorkspaceSchemaTemplate.vue"),
        props: true,
      },
      {
        path: "gitops",
        name: "setting.workspace.gitops",
        meta: { title: () => t("settings.sidebar.gitops") },
        component: () => import("@/views/SettingWorkspaceVCS.vue"),
        props: true,
      },
      {
        path: "gitops/new",
        name: "setting.workspace.gitops.create",
        meta: { title: () => t("repository.add-git-provider") },
        component: () => import("@/views/SettingWorkspaceVCSCreate.vue"),
        props: true,
      },
      {
        path: "gitops/:vcsSlug",
        name: "setting.workspace.gitops.detail",
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.vcsSlug as string;
            return useVCSV1Store().getVCSByUid(idFromSlug(slug))?.title ?? "";
          },
        },
        component: () => import("@/views/SettingWorkspaceVCSDetail.vue"),
        props: true,
      },
      {
        path: "mail-delivery",
        name: "setting.workspace.mail-delivery",
        meta: { title: () => t("settings.sidebar.mail-delivery") },
        component: () => import("@/views/SettingWorkspaceMailDelivery.vue"),
      },
      {
        path: "subscription",
        name: "setting.workspace.subscription",
        meta: { title: () => t("settings.sidebar.subscription") },
        component: () => import("@/views/SettingWorkspaceSubscription.vue"),
        props: true,
      },
      {
        path: "sql-review",
        name: "setting.workspace.sql-review",
        meta: {
          title: () => t("sql-review.title"),
        },
        component: () => import("@/views/SettingWorkspaceSQLReview.vue"),
        props: true,
      },
      {
        path: "sql-review/new",
        name: "setting.workspace.sql-review.create",
        meta: {
          title: () => t("sql-review.create.breadcrumb"),
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewCreate.vue"),
        props: true,
      },
      {
        path: "sql-review/:sqlReviewPolicySlug",
        name: "setting.workspace.sql-review.detail",
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.sqlReviewPolicySlug as string;
            return (
              useSQLReviewStore().getReviewPolicyByEnvironmentUID(
                String(idFromSlug(slug))
              )?.name ?? ""
            );
          },
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewDetail.vue"),
        props: true,
      },
      {
        path: "audit-log",
        name: "setting.workspace.audit-log",
        meta: {
          title: () => t("settings.sidebar.audit-log"),
        },
        component: () => import("@/views/SettingWorkspaceAuditLog.vue"),
        props: true,
      },
      {
        path: "debug-log",
        name: "setting.workspace.debug-log",
        meta: {
          title: () => t("settings.sidebar.debug-log"),
        },
        component: () => import("@/views/SettingWorkspaceDebugLog.vue"),
        props: true,
      },
      {
        path: "archive",
        name: "setting.workspace.archive",
        meta: { title: () => t("common.archived") },
        component: () => import("@/views/Archive.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceSettingRoutes;
