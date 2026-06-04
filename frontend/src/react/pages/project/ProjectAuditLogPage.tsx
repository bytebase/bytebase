import { useMemo } from "react";
import { AuditLogTable } from "@/react/components/AuditLogTable";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

export function ProjectAuditLogPage({ projectId }: { projectId: string }) {
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);

  const canExport = project
    ? hasProjectPermissionV2(project, "bb.auditLogs.export")
    : false;

  const readonlyScopes = useMemo(
    () => [{ id: "project", value: projectId }],
    [projectId]
  );

  return (
    <div className="flex flex-col">
      <div className="mx-4 mb-2">
        <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
      </div>
      <AuditLogTable
        parent={projectName}
        canExport={canExport}
        readonlyScopes={readonlyScopes}
      />
    </div>
  );
}
