import { AuditLogTable } from "@/components/AuditLogTable";
import { FeatureAttention } from "@/components/FeatureAttention";
import {
  WorkspacePageContent,
  WorkspacePageLayout,
} from "@/components/WorkspacePageLayout";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function AuditLogPage() {
  const canExport = hasWorkspacePermissionV2("bb.auditLogs.export");

  return (
    <WorkspacePageLayout className="gap-y-2">
      <FeatureAttention feature={PlanFeature.FEATURE_AUDIT_LOG} />
      <WorkspacePageContent>
        <AuditLogTable parent={`${projectNamePrefix}-`} canExport={canExport} />
      </WorkspacePageContent>
    </WorkspacePageLayout>
  );
}
