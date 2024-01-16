import { startCase } from "lodash-es";
import { RouteLocationNormalized, RouteRecordRaw } from "vue-router";
import { t } from "@/plugins/i18n";
import {
  useIdentityProviderStore,
  useSQLReviewStore,
  useVCSV1Store,
} from "@/store";
import { uidFromSlug } from "@/utils";
import SettingSidebar from "@/views/SettingSidebar.vue";

export const SETTING_ROUTE = "setting";
export const SETTING_ROUTE_WORKSPACE = `${SETTING_ROUTE}.workspace`;
export const SETTING_ROUTE_PROFILE = `${SETTING_ROUTE}.profile`;
export const SETTING_ROUTE_PROFILE_TWO_FACTOR = `${SETTING_ROUTE_PROFILE}.two-factor`;
export const SETTING_ROUTE_WORKSPACE_GENERAL = `${SETTING_ROUTE_WORKSPACE}.general`;
export const SETTING_ROUTE_WORKSPACE_MEMBER = `${SETTING_ROUTE_WORKSPACE}.member`;
export const SETTING_ROUTE_WORKSPACE_ROLE = `${SETTING_ROUTE_WORKSPACE}.role`;
export const SETTING_ROUTE_WORKSPACE_SSO = `${SETTING_ROUTE_WORKSPACE}.sso`;
export const SETTING_ROUTE_WORKSPACE_SSO_CREATE = `${SETTING_ROUTE_WORKSPACE_SSO}.create`;
export const SETTING_ROUTE_WORKSPACE_SSO_DETAIL = `${SETTING_ROUTE_WORKSPACE_SSO}.detail`;
export const SETTING_ROUTE_WORKSPACE_SENSITIVE_DATA = `${SETTING_ROUTE_WORKSPACE}.sensitive-data`;
export const SETTING_ROUTE_WORKSPACE_ACCESS_CONTROL = `${SETTING_ROUTE_WORKSPACE}.access-control`;
export const SETTING_ROUTE_WORKSPACE_RISK_CENTER = `${SETTING_ROUTE_WORKSPACE}.risk-center`;
export const SETTING_ROUTE_WORKSPACE_CUSTOM_APPROVAL = `${SETTING_ROUTE_WORKSPACE}.custom-approval`;
export const SETTING_ROUTE_WORKSPACE_SLOW_QUERY = `${SETTING_ROUTE_WORKSPACE}.slow-query`;
export const SETTING_ROUTE_WORKSPACE_SCHEMA_TEMPLATE = `${SETTING_ROUTE_WORKSPACE}.schema-template`;
export const SETTING_ROUTE_WORKSPACE_GITOPS = `${SETTING_ROUTE_WORKSPACE}.gitops`;
export const SETTING_ROUTE_WORKSPACE_GITOPS_CREATE = `${SETTING_ROUTE_WORKSPACE_GITOPS}.create`;
export const SETTING_ROUTE_WORKSPACE_GITOPS_DETAIL = `${SETTING_ROUTE_WORKSPACE_GITOPS}.detail`;
export const SETTING_ROUTE_WORKSPACE_MAIL_DELIVERY = `${SETTING_ROUTE_WORKSPACE}.mail-delivery`;
export const SETTING_ROUTE_WORKSPACE_SUBSCRIPTION = `${SETTING_ROUTE_WORKSPACE}.subscription`;
export const SETTING_ROUTE_WORKSPACE_SQL_REVIEW = `${SETTING_ROUTE_WORKSPACE}.sql-review`;
export const SETTING_ROUTE_WORKSPACE_SQL_REVIEW_CREATE = `${SETTING_ROUTE_WORKSPACE_SQL_REVIEW}.create`;
export const SETTING_ROUTE_WORKSPACE_SQL_REVIEW_DETAIL = `${SETTING_ROUTE_WORKSPACE_SQL_REVIEW}.detail`;
export const SETTING_ROUTE_WORKSPACE_AUDIT_LOG = `${SETTING_ROUTE_WORKSPACE}.audit-log`;
export const SETTING_ROUTE_WORKSPACE_DEBUG_LOG = `${SETTING_ROUTE_WORKSPACE}.debug-log`;
export const SETTING_ROUTE_WORKSPACE_ARCHIVE = `${SETTING_ROUTE_WORKSPACE}.archive`;

const workspaceSettingRoutes: RouteRecordRaw[] = [
  {
    path: "setting",
    name: SETTING_ROUTE_WORKSPACE,
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
        path: "profile",
        name: SETTING_ROUTE_PROFILE,
        meta: { title: () => t("settings.sidebar.profile") },
        component: () => import("@/views/ProfileDashboard.vue"),
        props: true,
      },
      {
        path: "profile/two-factor",
        name: SETTING_ROUTE_PROFILE_TWO_FACTOR,
        meta: { title: () => t("two-factor.self") },
        component: () => import("@/views/TwoFactorSetup.vue"),
        props: true,
      },
      {
        path: "general",
        name: SETTING_ROUTE_WORKSPACE_GENERAL,
        meta: {
          title: () => t("settings.sidebar.general"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceGeneral.vue"),
        props: true,
      },
      {
        path: "member",
        name: SETTING_ROUTE_WORKSPACE_MEMBER,
        meta: {
          title: () => t("settings.sidebar.members"),
          requiredWorkspacePermissionList: () => [
            "bb.policies.get",
            "bb.policies.update",
            "bb.settings.set",
          ],
        },
        component: () => import("@/views/SettingWorkspaceMember.vue"),
        props: true,
      },
      {
        path: "role",
        name: SETTING_ROUTE_WORKSPACE_ROLE,
        meta: {
          title: () => t("settings.sidebar.custom-roles"),
          requiredWorkspacePermissionList: () => ["bb.roles.list"],
        },
        component: () => import("@/views/SettingWorkspaceRole.vue"),
        props: true,
      },
      {
        path: "sso",
        name: SETTING_ROUTE_WORKSPACE_SSO,
        meta: {
          title: () => t("settings.sidebar.sso"),
          requiredWorkspacePermissionList: () => [
            "bb.identityProviders.create",
          ],
        },
        component: () => import("@/views/SettingWorkspaceSSO.vue"),
      },
      {
        path: "sso/new",
        name: SETTING_ROUTE_WORKSPACE_SSO_CREATE,
        meta: {
          title: () => t("settings.sidebar.sso"),
          requiredWorkspacePermissionList: () => [
            "bb.identityProviders.create",
          ],
        },
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
      },
      {
        path: "sso/:ssoName",
        name: SETTING_ROUTE_WORKSPACE_SSO_DETAIL,
        meta: {
          title: (route: RouteLocationNormalized) => {
            const name = route.params.ssoName as string;
            return (
              useIdentityProviderStore().getIdentityProviderByName(name)
                ?.title || t("settings.sidebar.sso")
            );
          },
          requiredWorkspacePermissionList: () => ["bb.identityProviders.get"],
        },
        component: () => import("@/views/SettingWorkspaceSSODetail.vue"),
        props: true,
      },
      {
        path: "sensitive-data",
        name: SETTING_ROUTE_WORKSPACE_SENSITIVE_DATA,
        meta: {
          title: () => t("settings.sidebar.sensitive-data"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSensitiveData.vue"),
        props: true,
      },
      {
        path: "access-control",
        name: SETTING_ROUTE_WORKSPACE_ACCESS_CONTROL,
        meta: {
          title: () => t("settings.sidebar.access-control"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceAccessControl.vue"),
        props: true,
      },
      {
        path: "risk-center",
        name: SETTING_ROUTE_WORKSPACE_RISK_CENTER,
        meta: {
          title: () => t("custom-approval.risk.risk-center"),
          requiredWorkspacePermissionList: () => ["bb.risks.list"],
        },
        component: () => import("@/views/SettingWorkspaceRiskCenter.vue"),
        props: true,
      },
      {
        path: "custom-approval",
        name: SETTING_ROUTE_WORKSPACE_CUSTOM_APPROVAL,
        meta: {
          title: () => t("custom-approval.self"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceCustomApproval.vue"),
        props: true,
      },
      {
        path: "slow-query",
        name: SETTING_ROUTE_WORKSPACE_SLOW_QUERY,
        meta: {
          title: () => startCase(t("slow-query.self")),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceSlowQuery.vue"),
        props: true,
      },
      {
        path: "schema-template",
        name: SETTING_ROUTE_WORKSPACE_SCHEMA_TEMPLATE,
        meta: {
          title: () => startCase(t("schema-template.self")),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSchemaTemplate.vue"),
        props: true,
      },
      {
        path: "gitops",
        name: SETTING_ROUTE_WORKSPACE_GITOPS,
        meta: {
          title: () => t("settings.sidebar.gitops"),
          requiredWorkspacePermissionList: () => [
            "bb.externalVersionControls.list",
          ],
        },
        component: () => import("@/views/SettingWorkspaceVCS.vue"),
        props: true,
      },
      {
        path: "gitops/new",
        name: SETTING_ROUTE_WORKSPACE_GITOPS_CREATE,
        meta: {
          title: () => t("repository.add-git-provider"),
          requiredWorkspacePermissionList: () => [
            "bb.externalVersionControls.create",
          ],
        },
        component: () => import("@/views/SettingWorkspaceVCSCreate.vue"),
        props: true,
      },
      {
        path: "gitops/:vcsSlug",
        name: SETTING_ROUTE_WORKSPACE_GITOPS_DETAIL,
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.vcsSlug as string;
            return useVCSV1Store().getVCSByUid(uidFromSlug(slug))?.title ?? "";
          },
          requiredWorkspacePermissionList: () => [
            "bb.externalVersionControls.get",
          ],
        },
        component: () => import("@/views/SettingWorkspaceVCSDetail.vue"),
        props: true,
      },
      {
        path: "mail-delivery",
        name: SETTING_ROUTE_WORKSPACE_MAIL_DELIVERY,
        meta: {
          title: () => t("settings.sidebar.mail-delivery"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceMailDelivery.vue"),
      },
      {
        path: "subscription",
        name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
        meta: {
          title: () => t("settings.sidebar.subscription"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSubscription.vue"),
        props: true,
      },
      {
        path: "sql-review",
        name: SETTING_ROUTE_WORKSPACE_SQL_REVIEW,
        meta: {
          title: () => t("sql-review.title"),
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReview.vue"),
        props: true,
      },
      {
        path: "sql-review/new",
        name: SETTING_ROUTE_WORKSPACE_SQL_REVIEW_CREATE,
        meta: {
          title: () => t("sql-review.create.breadcrumb"),
          requiredWorkspacePermissionList: () => ["bb.policies.create"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewCreate.vue"),
        props: true,
      },
      {
        path: "sql-review/:sqlReviewPolicySlug",
        name: SETTING_ROUTE_WORKSPACE_SQL_REVIEW_DETAIL,
        meta: {
          title: (route: RouteLocationNormalized) => {
            const slug = route.params.sqlReviewPolicySlug as string;
            return (
              useSQLReviewStore().getReviewPolicyByEnvironmentId(
                String(uidFromSlug(slug))
              )?.name ?? ""
            );
          },
          requiredWorkspacePermissionList: () => ["bb.policies.get"],
        },
        component: () => import("@/views/SettingWorkspaceSQLReviewDetail.vue"),
        props: true,
      },
      {
        path: "audit-log",
        name: SETTING_ROUTE_WORKSPACE_AUDIT_LOG,
        meta: {
          title: () => t("settings.sidebar.audit-log"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceAuditLog.vue"),
        props: true,
      },
      {
        path: "debug-log",
        name: SETTING_ROUTE_WORKSPACE_DEBUG_LOG,
        meta: {
          title: () => t("settings.sidebar.debug-log"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/SettingWorkspaceDebugLog.vue"),
        props: true,
      },
      {
        path: "archive",
        name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
        meta: {
          title: () => t("common.archived"),
          requiredWorkspacePermissionList: () => ["bb.settings.get"],
        },
        component: () => import("@/views/Archive.vue"),
        props: true,
      },
    ],
  },
];

export default workspaceSettingRoutes;
