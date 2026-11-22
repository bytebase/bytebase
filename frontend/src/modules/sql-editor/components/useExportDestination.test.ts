import { describe, expect, test, vi } from "vitest";
import { ExportDestination } from "@/types/proto-es/v1/common_pb";

vi.mock("@/hooks/useAppState", () => ({ usePlanFeature: () => true }));
vi.mock("@/hooks/useProjectByName", () => ({ useProjectByName: vi.fn() }));
vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorQueryDataPolicy: vi.fn(),
}));
vi.mock("@/stores/app", () => ({ useAppStore: vi.fn() }));

import { resolveExportDestination } from "./useExportDestination";

const url = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents";

describe("resolveExportDestination", () => {
  test("defaults to SharePoint and keeps download as the alternative", () => {
    expect(
      resolveExportDestination({
        hasSharePointFeature: true,
        sharepointUrl: url,
        disableDownload: false,
      })
    ).toEqual({
      destinations: [ExportDestination.SHAREPOINT, ExportDestination.DOWNLOAD],
      sharepointUrl: url,
    });
  });

  test("offers only SharePoint when download is disabled", () => {
    expect(
      resolveExportDestination({
        hasSharePointFeature: true,
        sharepointUrl: url,
        disableDownload: true,
      })
    ).toEqual({
      destinations: [ExportDestination.SHAREPOINT],
      sharepointUrl: url,
    });
  });

  test("offers only download without a SharePoint destination or the feature", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDestination({ ...input, disableDownload: false })
      ).toEqual({ destinations: [ExportDestination.DOWNLOAD] });
    }
  });

  test("offers nothing when download is disabled and SharePoint is not available", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDestination({ ...input, disableDownload: true })
      ).toEqual({ destinations: [] });
    }
  });
});
