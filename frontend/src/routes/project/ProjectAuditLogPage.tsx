import { useMemo } from "react";
import { AuditLogTable } from "@/components/AuditLogTable";
import { FeatureAttention } from "@/components/FeatureAttention";
import {
  ProjectPageContent,
  ProjectPageLayout,
} from "@/components/ProjectPageLayout";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
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
    <ProjectPageLayout className="gap-y-2">
      <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
      <ProjectPageContent>
        <AuditLogTable
          parent={projectName}
          canExport={canExport}
          readonlyScopes={readonlyScopes}
        />
      </ProjectPageContent>
    </ProjectPageLayout>
  );
}
