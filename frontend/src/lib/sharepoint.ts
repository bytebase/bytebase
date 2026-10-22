import type { SharePointLocation } from "@/types/proto-es/v1/project_service_pb";

// Formats a resolved SharePoint location as "Site / Library / Folder".
export function formatSharePointLocation(location: SharePointLocation) {
  return [location.siteName, location.libraryName, location.folderPath]
    .filter(Boolean)
    .join(" / ");
}
