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

const both = [ExportDestination.DOWNLOAD, ExportDestination.SHAREPOINT];

describe("resolveExportDestination", () => {
  test("defaults to SharePoint and keeps download as the alternative", () => {
    expect(
      resolveExportDestination({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDestinations: both,
      })
    ).toEqual({
      destinations: [ExportDestination.SHAREPOINT, ExportDestination.DOWNLOAD],
      sharepointUrl: url,
    });
  });

  test("offers only SharePoint when the policy allows only SharePoint", () => {
    expect(
      resolveExportDestination({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDestinations: [ExportDestination.SHAREPOINT],
      })
    ).toEqual({
      destinations: [ExportDestination.SHAREPOINT],
      sharepointUrl: url,
    });
  });

  test("offers only download when the policy allows only download", () => {
    expect(
      resolveExportDestination({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDestinations: [ExportDestination.DOWNLOAD],
      })
    ).toEqual({ destinations: [ExportDestination.DOWNLOAD] });
  });

  test("offers only download without a SharePoint destination or the feature", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDestination({ ...input, allowedDestinations: both })
      ).toEqual({ destinations: [ExportDestination.DOWNLOAD] });
    }
  });

  test("offers nothing when the policy allows only SharePoint and it is not configured", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDestination({
          ...input,
          allowedDestinations: [ExportDestination.SHAREPOINT],
        })
      ).toEqual({ destinations: [] });
    }
  });
});
