import { useEffect } from "react";
import { usePlanFeature } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useSQLEditorQueryDataPolicy } from "@/modules/sql-editor/hooks/useSQLEditorState";
import { useAppStore } from "@/stores/app";
import { ExportDestination } from "@/types/proto-es/v1/common_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

export interface ExportDestinationAvailability {
  // The offered destinations; the first is the default. Empty means the caller
  // hides export.
  destinations: ExportDestination[];
  // Set only when SharePoint is offered.
  sharepointUrl?: string;
}

// An export access grant never adds a destination: Download is offered unless
// the workspace disables it, and SharePoint only when the project has a
// SharePoint folder.
export function resolveExportDestination({
  hasSharePointFeature,
  sharepointUrl,
  disableDownload,
}: {
  hasSharePointFeature: boolean;
  sharepointUrl: string;
  disableDownload: boolean;
}): ExportDestinationAvailability {
  const destinations: ExportDestination[] = [];
  const offersSharePoint = hasSharePointFeature && sharepointUrl !== "";
  if (offersSharePoint) {
    destinations.push(ExportDestination.SHAREPOINT);
  }
  if (!disableDownload) {
    destinations.push(ExportDestination.DOWNLOAD);
  }
  return offersSharePoint ? { destinations, sharepointUrl } : { destinations };
}

export function useExportDestination(projectName: string) {
  const project = useProjectByName(projectName);
  const hasSharePointFeature = usePlanFeature(
    PlanFeature.FEATURE_SHAREPOINT_EXPORT
  );
  // The server enforces disable_download without a license check, so it is
  // read regardless of FEATURE_QUERY_POLICY.
  const { disableDownload } = useSQLEditorQueryDataPolicy(projectName);

  useEffect(() => {
    if (!projectName) return;
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName, true)
      .catch(() => undefined);
  }, [projectName]);

  return resolveExportDestination({
    hasSharePointFeature,
    sharepointUrl: project.exportSetting?.sharepoint?.url ?? "",
    disableDownload,
  });
}

export type ExportDestinationState = ReturnType<typeof useExportDestination>;
