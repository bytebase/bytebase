// Route-name constants for the React router. Kept VUE-FREE: the only re-export
// is from `@/router/dashboard/workspaceRoutes` (a pure-constant, import-free
// module). Everything else is inlined as literals, because the source
// vue-router modules eagerly import Vue SFCs / vue-i18n — re-exporting from them
// would pull the whole Vue graph into the React bundle and break the teardown
// phase. Values mirror the corresponding `@/router/**` modules; those are
// deleted in teardown, at which point this file becomes the single source.

import {
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
} from "@/router/dashboard/workspaceRoutes";

// workspaceRoutes.ts is import-free (no Vue), so re-exporting it is safe.
export {
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
  WORKSPACE_ROUTE_DATA_EXPORT,
  WORKSPACE_ROUTE_GLOBAL_MASKING,
  WORKSPACE_ROUTE_GROUPS,
  WORKSPACE_ROUTE_IDENTITY_PROVIDER_DETAIL,
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS,
  WORKSPACE_ROUTE_IM,
  WORKSPACE_ROUTE_INTEGRATION,
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
} from "@/router/dashboard/workspaceRoutes";

// --- auth.ts ---
export const AUTH_SIGNIN_MODULE = "auth.signin";
export const AUTH_SIGNIN_ADMIN_MODULE = "auth.signin.admin";
export const AUTH_SIGNUP_MODULE = "auth.signup";
export const AUTH_MFA_MODULE = "auth.mfa";
export const AUTH_PASSWORD_RESET_MODULE = "auth.password.reset";
export const AUTH_PASSWORD_FORGOT_MODULE = "auth.password.forgot";
export const AUTH_OAUTH_CALLBACK_MODULE = "auth.oauth.callback";
export const AUTH_OIDC_CALLBACK_MODULE = "auth.oidc.callback";
export const AUTH_PROFILE_SETUP_MODULE = "auth.profile.setup";
export const AUTH_2FA_SETUP_MODULE = "auth.2fa.setup";
export const OAUTH2_CONSENT_MODULE = "oauth2.consent";

// --- setup.ts ---
export const SETUP_MODULE = "setup";

// --- sqlEditor.ts ---
export const SQL_EDITOR_HOME_MODULE = "sql-editor.home";
export const SQL_EDITOR_PROJECT_MODULE = "sql-editor.project";
export const SQL_EDITOR_INSTANCE_MODULE = "sql-editor.instance";
export const SQL_EDITOR_DATABASE_MODULE = "sql-editor.database";
export const SQL_EDITOR_WORKSHEET_MODULE = "sql-editor.worksheet";

// --- dashboard/workspaceSetting.ts ---
export const SETTING_ROUTE = "setting";
export const SETTING_ROUTE_WORKSPACE = `${SETTING_ROUTE}.workspace`;
export const SETTING_ROUTE_PROFILE = `${SETTING_ROUTE}.profile`;
export const SETTING_ROUTE_PROFILE_TWO_FACTOR = `${SETTING_ROUTE_PROFILE}.two-factor`;
export const SETTING_ROUTE_WORKSPACE_GENERAL = `${SETTING_ROUTE_WORKSPACE}.general`;
export const SETTING_ROUTE_WORKSPACE_SUBSCRIPTION = `${SETTING_ROUTE_WORKSPACE}.subscription`;

// --- dashboard/instance.ts ---
export const INSTANCE_ROUTE_CREATE = `${INSTANCE_ROUTE_DASHBOARD}.create`;
export const INSTANCE_ROUTE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.detail`;
export const INSTANCE_ROUTE_DATABASE_DETAIL = `${INSTANCE_ROUTE_DASHBOARD}.database.detail`;

// --- dashboard/environmentV1.ts ---
export const ENVIRONMENT_V1_ROUTE_DETAIL = `${ENVIRONMENT_V1_ROUTE_DASHBOARD}.detail`;

// --- dashboard/projectV1.ts ---
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
export const PROJECT_V1_ROUTE_ISSUES = `${PROJECT_V1_ROUTE_DASHBOARD}.issue`;
export const PROJECT_V1_ROUTE_ISSUE_DETAIL = `${PROJECT_V1_ROUTE_ISSUES}.detail`;
export const PROJECT_V1_ROUTE_PLANS = `${PROJECT_V1_ROUTE_DASHBOARD}.plan`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.plan.detail`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS = `${PROJECT_V1_ROUTE_PLAN_DETAIL}.specs`;
export const PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL = `${PROJECT_V1_ROUTE_PLAN_DETAIL}.spec.detail`;
export const PROJECT_V1_ROUTE_SYNC_SCHEMA = `${PROJECT_V1_ROUTE_DASHBOARD}.sync-schema`;
export const PROJECT_V1_ROUTE_AUDIT_LOGS = `${PROJECT_V1_ROUTE_DASHBOARD}.audit-logs`;
export const PROJECT_V1_ROUTE_WEBHOOKS = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook`;
export const PROJECT_V1_ROUTE_WEBHOOK_CREATE = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.create`;
export const PROJECT_V1_ROUTE_WEBHOOK_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.webhook.detail`;
export const PROJECT_V1_ROUTE_ACCESS_GRANTS = `${PROJECT_V1_ROUTE_DASHBOARD}.access-grants`;
export const PROJECT_V1_ROUTE_MEMBERS = `${PROJECT_V1_ROUTE_DASHBOARD}.members`;
export const PROJECT_V1_ROUTE_SERVICE_ACCOUNTS = `${PROJECT_V1_ROUTE_DASHBOARD}.service-accounts`;
export const PROJECT_V1_ROUTE_WORKLOAD_IDENTITIES = `${PROJECT_V1_ROUTE_DASHBOARD}.workload-identities`;
export const PROJECT_V1_ROUTE_SETTINGS = `${PROJECT_V1_ROUTE_DASHBOARD}.settings`;
export const PROJECT_V1_ROUTE_DATA_EXPORT = `${PROJECT_V1_ROUTE_DASHBOARD}.data-export`;
export const PROJECT_V1_ROUTE_RELEASES = `${PROJECT_V1_ROUTE_DASHBOARD}.release`;
export const PROJECT_V1_ROUTE_RELEASE_DETAIL = `${PROJECT_V1_ROUTE_DASHBOARD}.release.detail`;
export const PROJECT_V1_ROUTE_ROLLOUTS = `${PROJECT_V1_ROUTE_DASHBOARD}.rollouts`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT = `${PROJECT_V1_ROUTE_ROLLOUTS}.rollout`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE = `${PROJECT_V1_ROUTE_PLAN_ROLLOUT}.stage`;
export const PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK = `${PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE}.task`;
export const PROJECT_V1_ROUTE_GITOPS = `${PROJECT_V1_ROUTE_DASHBOARD}.gitops`;

// --- dashboard/projectV1RouteHelpers.ts ---
export const PLAN_DETAIL_PHASE_CHANGES = "changes";
export const PLAN_DETAIL_PHASE_REVIEW = "review";
export const PLAN_DETAIL_PHASE_DEPLOY = "deploy";
