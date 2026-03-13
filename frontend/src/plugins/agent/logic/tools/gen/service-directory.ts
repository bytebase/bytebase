// LLM-generated service directory. Regenerate with:
//   pnpm --dir frontend run generate:service-directory
// See frontend/src/plugins/agent/AGENT.md for maintenance guide.

export const serviceDirectory = `API Directory — use search_api(service="...") to browse endpoints, search_api(operationId="...") for schemas.

Database Management:
- DatabaseService: CRUD databases, list schemas/tables/columns, metadata, secrets, changelogs, slow queries, backups
- DatabaseCatalogService: semantic types, column masking config, data classification
- DatabaseGroupService: database groups, matching databases by expression

SQL & Queries:
- SQLService: execute SQL queries, export query results, SQL check/lint, parse SQL, differencing
- SheetService: internal saved SQL sheets (system use)
- WorksheetService: user-facing saved SQL worksheets

Change Management:
- PlanService: create/update database change plans, plan checks, spec management
- IssueService: change tickets (issues), approval flow, issue status, comments, subscribers
- RolloutService: rollout stages, tasks, task runs, batch rollout execution
- RevisionService: schema revision snapshots
- ReleaseService: release bundles grouping multiple changes

Access & Identity:
- AuthService: login, logout, password management, OAuth/OIDC
- UserService: user CRUD, list users
- GroupService: user groups
- ServiceAccountService: service account CRUD
- WorkloadIdentityService: workload identity federation
- RoleService: custom RBAC role definitions
- IdentityProviderService: SSO/IdP config (OIDC, OAuth2, LDAP)
- AccessGrantService: time-limited data access grants, approval

Infrastructure:
- InstanceService: database server instances, connections, SSL config
- InstanceRoleService: database roles/users on instances
- EnvironmentService: environments (dev, staging, prod)

Policy & Compliance:
- OrgPolicyService: organization policies (masking, access control, backup, SQL review)
- ReviewConfigService: SQL review rule configuration
- RiskService: risk rules for custom approval routing

Workspace:
- ProjectService: projects, webhooks, Git/VCS connectors, IAM bindings, project search
- SettingService: workspace-level settings (branding, approval, classification, mail, etc.)
- WorkspaceService: workspace metadata
- SubscriptionService: license and subscription info
- ActuatorService: server health, version, debug info
- AuditLogService: workspace audit logs

Utility:
- CelService: parse/format CEL expressions (Common Expression Language)
- AIService: internal AI chat endpoint (system use)
`;
