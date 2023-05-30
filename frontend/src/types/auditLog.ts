export enum AuditActivityType {
  // Workspace level.
  //
  // Member related
  WorkspaceMemberCreate = "bb.member.create",
  WorkspaceMemberRoleUpdate = "bb.member.role.update",
  WorkspaceMemberActivate = "bb.member.activate",
  WorkspaceMemberDeactivate = "bb.member.deactivate",

  // Project level.
  //
  // Project related
  DatabaseTransfer = "bb.project.database.transfer",
  ProjectMemberRoleUpdate = "bb.project.member.role.update",

  // SQL Editor related.
  SQLEditorQuery = "bb.sql-editor.query",
}

export enum AuditActivityLevel {
  INFO = "INFO",
  WARN = "WARN",
  ERROR = "ERROR",
}

// Mapping from an activity type to a human readable string.
export const AuditActivityTypeI18nNameMap: Record<AuditActivityType, string> = {
  [AuditActivityType.WorkspaceMemberCreate]:
    "audit-log.type.workspace.member-create",
  [AuditActivityType.WorkspaceMemberRoleUpdate]:
    "audit-log.type.workspace.member-role-update",
  [AuditActivityType.WorkspaceMemberActivate]:
    "audit-log.type.workspace.member-activate",
  [AuditActivityType.WorkspaceMemberDeactivate]:
    "audit-log.type.workspace.member-deactivate",
  [AuditActivityType.DatabaseTransfer]:
    "audit-log.type.project.database-transfer",
  [AuditActivityType.ProjectMemberRoleUpdate]:
    "audit-log.type.project.member-role-update",
  [AuditActivityType.SQLEditorQuery]: "audit-log.type.sql-editor-query",
};

export type AuditLog = {
  createdTs: number;
  creator: string;
  type: AuditActivityType;
  level: AuditActivityLevel;
  comment: string;
  payload: unknown;
};
