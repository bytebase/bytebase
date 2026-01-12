package iam

import "github.com/bytebase/bytebase/backend/common/permission"

// Re-export Permission type from shared package.
type Permission = permission.Permission

// Re-export all permission constants.
const (
	PermissionAuditLogsExport                      = permission.PermissionAuditLogsExport
	PermissionAuditLogsSearch                      = permission.PermissionAuditLogsSearch
	PermissionChangelogsGet                        = permission.PermissionChangelogsGet
	PermissionChangelogsList                       = permission.PermissionChangelogsList
	PermissionDatabaseCatalogsGet                  = permission.PermissionDatabaseCatalogsGet
	PermissionDatabaseCatalogsUpdate               = permission.PermissionDatabaseCatalogsUpdate
	PermissionDatabaseGroupsCreate                 = permission.PermissionDatabaseGroupsCreate
	PermissionDatabaseGroupsDelete                 = permission.PermissionDatabaseGroupsDelete
	PermissionDatabaseGroupsGet                    = permission.PermissionDatabaseGroupsGet
	PermissionDatabaseGroupsList                   = permission.PermissionDatabaseGroupsList
	PermissionDatabaseGroupsUpdate                 = permission.PermissionDatabaseGroupsUpdate
	PermissionDatabasesCheck                       = permission.PermissionDatabasesCheck
	PermissionDatabasesGet                         = permission.PermissionDatabasesGet
	PermissionDatabasesGetSchema                   = permission.PermissionDatabasesGetSchema
	PermissionDatabasesList                        = permission.PermissionDatabasesList
	PermissionDatabasesSync                        = permission.PermissionDatabasesSync
	PermissionDatabasesUpdate                      = permission.PermissionDatabasesUpdate
	PermissionIdentityProvidersCreate              = permission.PermissionIdentityProvidersCreate
	PermissionIdentityProvidersDelete              = permission.PermissionIdentityProvidersDelete
	PermissionIdentityProvidersGet                 = permission.PermissionIdentityProvidersGet
	PermissionIdentityProvidersUpdate              = permission.PermissionIdentityProvidersUpdate
	PermissionInstancesCreate                      = permission.PermissionInstancesCreate
	PermissionInstancesDelete                      = permission.PermissionInstancesDelete
	PermissionInstancesGet                         = permission.PermissionInstancesGet
	PermissionInstancesList                        = permission.PermissionInstancesList
	PermissionInstancesSync                        = permission.PermissionInstancesSync
	PermissionInstancesUndelete                    = permission.PermissionInstancesUndelete
	PermissionInstancesUpdate                      = permission.PermissionInstancesUpdate
	PermissionInstanceRolesGet                     = permission.PermissionInstanceRolesGet
	PermissionInstanceRolesList                    = permission.PermissionInstanceRolesList
	PermissionIssueCommentsCreate                  = permission.PermissionIssueCommentsCreate
	PermissionIssueCommentsList                    = permission.PermissionIssueCommentsList
	PermissionIssueCommentsUpdate                  = permission.PermissionIssueCommentsUpdate
	PermissionIssuesCreate                         = permission.PermissionIssuesCreate
	PermissionIssuesGet                            = permission.PermissionIssuesGet
	PermissionIssuesList                           = permission.PermissionIssuesList
	PermissionIssuesUpdate                         = permission.PermissionIssuesUpdate
	PermissionPlanCheckRunsGet                     = permission.PermissionPlanCheckRunsGet
	PermissionPlanCheckRunsRun                     = permission.PermissionPlanCheckRunsRun
	PermissionPlansCreate                          = permission.PermissionPlansCreate
	PermissionPlansGet                             = permission.PermissionPlansGet
	PermissionPlansList                            = permission.PermissionPlansList
	PermissionPlansUpdate                          = permission.PermissionPlansUpdate
	PermissionPoliciesCreate                       = permission.PermissionPoliciesCreate
	PermissionPoliciesDelete                       = permission.PermissionPoliciesDelete
	PermissionPoliciesGet                          = permission.PermissionPoliciesGet
	PermissionPoliciesList                         = permission.PermissionPoliciesList
	PermissionPoliciesUpdate                       = permission.PermissionPoliciesUpdate
	PermissionPoliciesGetMaskingRulePolicy         = permission.PermissionPoliciesGetMaskingRulePolicy
	PermissionPoliciesUpdateMaskingRulePolicy      = permission.PermissionPoliciesUpdateMaskingRulePolicy
	PermissionPoliciesCreateMaskingRulePolicy      = permission.PermissionPoliciesCreateMaskingRulePolicy
	PermissionPoliciesDeleteMaskingRulePolicy      = permission.PermissionPoliciesDeleteMaskingRulePolicy
	PermissionPoliciesGetMaskingExemptionPolicy    = permission.PermissionPoliciesGetMaskingExemptionPolicy
	PermissionPoliciesUpdateMaskingExemptionPolicy = permission.PermissionPoliciesUpdateMaskingExemptionPolicy
	PermissionPoliciesCreateMaskingExemptionPolicy = permission.PermissionPoliciesCreateMaskingExemptionPolicy
	PermissionPoliciesDeleteMaskingExemptionPolicy = permission.PermissionPoliciesDeleteMaskingExemptionPolicy
	PermissionProjectsCreate                       = permission.PermissionProjectsCreate
	PermissionProjectsDelete                       = permission.PermissionProjectsDelete
	PermissionProjectsGet                          = permission.PermissionProjectsGet
	PermissionProjectsGetIAMPolicy                 = permission.PermissionProjectsGetIAMPolicy
	PermissionProjectsList                         = permission.PermissionProjectsList
	PermissionProjectsSetIAMPolicy                 = permission.PermissionProjectsSetIAMPolicy
	PermissionProjectsUndelete                     = permission.PermissionProjectsUndelete
	PermissionProjectsUpdate                       = permission.PermissionProjectsUpdate
	PermissionReleasesCheck                        = permission.PermissionReleasesCheck
	PermissionReleasesCreate                       = permission.PermissionReleasesCreate
	PermissionReleasesDelete                       = permission.PermissionReleasesDelete
	PermissionReleasesGet                          = permission.PermissionReleasesGet
	PermissionReleasesList                         = permission.PermissionReleasesList
	PermissionReleasesUndelete                     = permission.PermissionReleasesUndelete
	PermissionReleasesUpdate                       = permission.PermissionReleasesUpdate
	PermissionReviewConfigsCreate                  = permission.PermissionReviewConfigsCreate
	PermissionReviewConfigsDelete                  = permission.PermissionReviewConfigsDelete
	PermissionReviewConfigsGet                     = permission.PermissionReviewConfigsGet
	PermissionReviewConfigsList                    = permission.PermissionReviewConfigsList
	PermissionReviewConfigsUpdate                  = permission.PermissionReviewConfigsUpdate
	PermissionRevisionsCreate                      = permission.PermissionRevisionsCreate
	PermissionRevisionsDelete                      = permission.PermissionRevisionsDelete
	PermissionRevisionsGet                         = permission.PermissionRevisionsGet
	PermissionRevisionsList                        = permission.PermissionRevisionsList
	PermissionRolesCreate                          = permission.PermissionRolesCreate
	PermissionRolesDelete                          = permission.PermissionRolesDelete
	PermissionRolesList                            = permission.PermissionRolesList
	PermissionRolesGet                             = permission.PermissionRolesGet
	PermissionRolesUpdate                          = permission.PermissionRolesUpdate
	PermissionRolloutsCreate                       = permission.PermissionRolloutsCreate
	PermissionRolloutsGet                          = permission.PermissionRolloutsGet
	PermissionRolloutsList                         = permission.PermissionRolloutsList
	PermissionSettingsGet                          = permission.PermissionSettingsGet
	PermissionSettingsList                         = permission.PermissionSettingsList
	PermissionSettingsSet                          = permission.PermissionSettingsSet
	PermissionEnvironmentSettingsGet               = permission.PermissionEnvironmentSettingsGet
	PermissionEnvironmentSettingsSet               = permission.PermissionEnvironmentSettingsSet
	PermissionWorkspaceProfileSettingsGet          = permission.PermissionWorkspaceProfileSettingsGet
	PermissionWorkspaceProfileSettingsSet          = permission.PermissionWorkspaceProfileSettingsSet
	PermissionSheetsCreate                         = permission.PermissionSheetsCreate
	PermissionSheetsGet                            = permission.PermissionSheetsGet
	PermissionSheetsUpdate                         = permission.PermissionSheetsUpdate
	PermissionSQLSelect                            = permission.PermissionSQLSelect
	PermissionSQLDdl                               = permission.PermissionSQLDdl
	PermissionSQLDml                               = permission.PermissionSQLDml
	PermissionSQLExplain                           = permission.PermissionSQLExplain
	PermissionSQLInfo                              = permission.PermissionSQLInfo
	PermissionSQLAdmin                             = permission.PermissionSQLAdmin
	PermissionTaskRunsCreate                       = permission.PermissionTaskRunsCreate
	PermissionTaskRunsList                         = permission.PermissionTaskRunsList
	PermissionGroupsCreate                         = permission.PermissionGroupsCreate
	PermissionGroupsDelete                         = permission.PermissionGroupsDelete
	PermissionGroupsGet                            = permission.PermissionGroupsGet
	PermissionGroupsList                           = permission.PermissionGroupsList
	PermissionGroupsUpdate                         = permission.PermissionGroupsUpdate
	PermissionUsersCreate                          = permission.PermissionUsersCreate
	PermissionUsersDelete                          = permission.PermissionUsersDelete
	PermissionUsersGet                             = permission.PermissionUsersGet
	PermissionUsersList                            = permission.PermissionUsersList
	PermissionUsersUndelete                        = permission.PermissionUsersUndelete
	PermissionUsersUpdate                          = permission.PermissionUsersUpdate
	PermissionUsersUpdateEmail                     = permission.PermissionUsersUpdateEmail
	PermissionWorksheetsGet                        = permission.PermissionWorksheetsGet
	PermissionWorksheetsManage                     = permission.PermissionWorksheetsManage
	PermissionWorkspacesGetIamPolicy               = permission.PermissionWorkspacesGetIamPolicy
	PermissionWorkspacesSetIamPolicy               = permission.PermissionWorkspacesSetIamPolicy
)

// Re-export functions. Note: shared package uses Exist/Exists (without Permission prefix).
var (
	PermissionsExist = permission.Exists
	PermissionExist  = permission.Exist
)
