import { AuditLogTable } from "@/react/components/AuditLogTable";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

export function AuditLogPage() {
  const canExport = hasWorkspacePermissionV2("bb.auditLogs.export");

  return (
    <AuditLogTable parent={`${projectNamePrefix}-`} canExport={canExport} />
  );
}
