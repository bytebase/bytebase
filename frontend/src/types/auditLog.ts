import { AuditLog_Severity } from "./proto-es/v1/audit_log_service_pb";

export interface AuditLogFilter {
  method?: string;
  level?: AuditLog_Severity;
  userEmail?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
}

export interface SearchAuditLogsParams {
  parent?: string;
  filter: AuditLogFilter;
  pageSize?: number;
  pageToken?: string;
  orderBy?: string;
}
