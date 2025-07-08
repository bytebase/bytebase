import { AuditLog_Severity } from "./proto-es/v1/audit_log_service_pb";

export interface SearchAuditLogsParams {
  parent?: string;
  method?: string;
  level?: AuditLog_Severity;
  userEmail?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  order?: "asc" | "desc";
  pageSize?: number;
  pageToken?: string;
}
