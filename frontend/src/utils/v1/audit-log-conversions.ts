import { AuditLog_Severity } from "@/types/proto-es/v1/audit_log_service_pb";

/**
 * Converts AuditLog_Severity enum value to its string representation
 * @param severity - The severity enum value
 * @returns The string name of the severity level (e.g., "INFO", "ERROR", "WARNING")
 */
export function severityToString(severity: AuditLog_Severity): string {
  return AuditLog_Severity[severity];
}