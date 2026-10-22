import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { usePlanFeature } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import type { SharePointLocation } from "@/types/proto-es/v1/project_service_pb";
import { ExportRequest_Delivery } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

export type ExportDeliveryUnavailableReason =
  | "NO_DESTINATION"
  | "FEATURE_UNAVAILABLE";

export interface ExportDeliveryAvailability {
  // The offered deliveries; the first is the default.
  deliveries: ExportRequest_Delivery[];
  destination?: SharePointLocation;
  unavailableReason?: ExportDeliveryUnavailableReason;
}

export function resolveExportDelivery({
  hasSharePointFeature,
  sharepointUrl,
  destination,
  localDownloadDisabled,
}: {
  hasSharePointFeature: boolean;
  sharepointUrl: string;
  destination?: SharePointLocation;
  localDownloadDisabled: boolean;
}): ExportDeliveryAvailability {
  const deliveries: ExportRequest_Delivery[] = [];
  const canDeliverToSharePoint = hasSharePointFeature && sharepointUrl !== "";
  if (canDeliverToSharePoint) {
    deliveries.push(ExportRequest_Delivery.SHAREPOINT);
  }
  if (!localDownloadDisabled) {
    deliveries.push(ExportRequest_Delivery.LOCAL);
  }
  if (deliveries.length === 0) {
    return {
      deliveries,
      unavailableReason: sharepointUrl
        ? "FEATURE_UNAVAILABLE"
        : "NO_DESTINATION",
    };
  }
  return {
    deliveries,
    destination: canDeliverToSharePoint ? destination : undefined,
  };
}

export function useExportDelivery(projectName: string) {
  const { t } = useTranslation();
  const project = useProjectByName(projectName);
  const hasSharePointFeature = usePlanFeature(
    PlanFeature.FEATURE_SHAREPOINT_EXPORT
  );
  const localDownloadDisabled = useAppStore(
    (s) => s.getWorkspaceProfile().disableLocalExportDownload
  );

  useEffect(() => {
    if (!projectName) return;
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName, true)
      .catch(() => undefined);
  }, [projectName]);

  const availability = resolveExportDelivery({
    hasSharePointFeature,
    sharepointUrl: project.exportDestination?.sharepointUrl ?? "",
    destination: project.exportDestination?.sharepointLocation,
    localDownloadDisabled,
  });
  const unavailableMessage =
    availability.unavailableReason === "NO_DESTINATION"
      ? t("export-data.unavailable.no-destination")
      : availability.unavailableReason === "FEATURE_UNAVAILABLE"
        ? t("export-data.unavailable.feature-unavailable")
        : undefined;
  return { ...availability, unavailableMessage };
}

export type ExportDeliveryState = ReturnType<typeof useExportDelivery>;
