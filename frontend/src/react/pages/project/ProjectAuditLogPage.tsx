import { useMemo } from "react";
import { AuditLogTable } from "@/react/components/AuditLogTable";
import { useVueState } from "@/react/hooks/useVueState";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

export function ProjectAuditLogPage({ projectId }: { projectId: string }) {
  const projectName = `${projectNamePrefix}${projectId}`;
  const projectStore = useProjectV1Store();
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const canExport = project
    ? hasProjectPermissionV2(project, "bb.auditLogs.export")
    : false;

  const readonlyScopes = useMemo(
    () => [{ id: "project", value: projectId }],
    [projectId]
  );

  return (
    <AuditLogTable
      parent={projectName}
      canExport={canExport}
      readonlyScopes={readonlyScopes}
    />
  );
}
