import { useMemo } from "react";
import { AuditLogTable } from "@/react/components/AuditLogTable";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { useVueState } from "@/react/hooks/useVueState";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
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
