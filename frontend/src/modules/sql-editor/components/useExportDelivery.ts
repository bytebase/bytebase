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

// An export access grant never adds a delivery: a delivery is offered only when
// the workspace query data policy allows it and, for a destination, the
// project has one configured.
export function resolveExportDelivery({
  hasSharePointFeature,
  sharepointUrl,
  allowedDeliveries,
}: {
  hasSharePointFeature: boolean;
  sharepointUrl: string;
  allowedDeliveries: readonly ExportDelivery[];
}): ExportDeliveryAvailability {
  const deliveries: ExportDelivery[] = [];
  const canDeliverToSharePoint =
    allowedDeliveries.includes(ExportDelivery.SHAREPOINT) &&
    hasSharePointFeature &&
    sharepointUrl !== "";
  if (canDeliverToSharePoint) {
    deliveries.push(ExportDelivery.SHAREPOINT);
  }
  if (allowedDeliveries.includes(ExportDelivery.LOCAL)) {
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
  // The server enforces the allowed deliveries without a license check, so
  // they are read regardless of FEATURE_QUERY_POLICY.
  const { allowedExportDeliveries } = useSQLEditorQueryDataPolicy(projectName);

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
    allowedDeliveries: allowedExportDeliveries,
  });
}

export type ExportDeliveryState = ReturnType<typeof useExportDelivery>;
