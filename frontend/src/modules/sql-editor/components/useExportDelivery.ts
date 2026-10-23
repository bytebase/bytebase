import { useEffect } from "react";
import { usePlanFeature } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useSQLEditorQueryDataPolicy } from "@/modules/sql-editor/hooks/useSQLEditorState";
import { useAppStore } from "@/stores/app";
import { ExportDelivery } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

export interface ExportDeliveryAvailability {
  // The offered deliveries; the first is the default. Empty means the caller
  // hides export.
  deliveries: ExportDelivery[];
  // Set only when SharePoint is offered.
  sharepointUrl?: string;
}

// An export access grant never adds a delivery: local download follows only
// the workspace's disableExport, and destinations need no grant.
export function resolveExportDelivery({
  hasSharePointFeature,
  sharepointUrl,
  disableExport,
}: {
  hasSharePointFeature: boolean;
  sharepointUrl: string;
  disableExport: boolean;
}): ExportDeliveryAvailability {
  const deliveries: ExportDelivery[] = [];
  const canDeliverToSharePoint = hasSharePointFeature && sharepointUrl !== "";
  if (canDeliverToSharePoint) {
    deliveries.push(ExportDelivery.SHAREPOINT);
  }
  if (!disableExport) {
    deliveries.push(ExportDelivery.LOCAL);
  }
  return canDeliverToSharePoint
    ? { deliveries, sharepointUrl }
    : { deliveries };
}

export function useExportDelivery(projectName: string) {
  const project = useProjectByName(projectName);
  const hasSharePointFeature = usePlanFeature(
    PlanFeature.FEATURE_SHAREPOINT_EXPORT
  );
  // The server enforces disableExport without a license check, so it is read
  // regardless of FEATURE_QUERY_POLICY.
  const { disableExport } = useSQLEditorQueryDataPolicy(projectName);

  useEffect(() => {
    if (!projectName) return;
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName, true)
      .catch(() => undefined);
  }, [projectName]);

  return resolveExportDelivery({
    hasSharePointFeature,
    sharepointUrl: project.exportSetting?.sharepoint?.url ?? "",
    disableExport,
  });
}

export type ExportDeliveryState = ReturnType<typeof useExportDelivery>;
