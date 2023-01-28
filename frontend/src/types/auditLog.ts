export enum AuditActivityType {
  // Member related
  MemberCreate = "bb.member.create",
  MemberRoleUpdate = "bb.member.role.update",
  MemberActivate = "bb.member.activate",
  MemberDeactivate = "bb.member.deactivate",

  // Project related
  DatabaseTransfer = "bb.project.database.transfer",

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
  [AuditActivityType.MemberCreate]: "audit-log.type.member-create",
  [AuditActivityType.MemberRoleUpdate]: "audit-log.type.member-role-update",
  [AuditActivityType.MemberActivate]: "audit-log.type.member-activate",
  [AuditActivityType.MemberDeactivate]: "audit-log.type.member-deactivate",
  [AuditActivityType.DatabaseTransfer]: "audit-log.type.database-transfer",
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
