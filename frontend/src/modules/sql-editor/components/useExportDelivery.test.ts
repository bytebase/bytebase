import { describe, expect, test, vi } from "vitest";
import { ExportDelivery } from "@/types/proto-es/v1/common_pb";

vi.mock("@/hooks/useAppState", () => ({ usePlanFeature: () => true }));
vi.mock("@/hooks/useProjectByName", () => ({ useProjectByName: vi.fn() }));
vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorQueryDataPolicy: vi.fn(),
}));
vi.mock("@/stores/app", () => ({ useAppStore: vi.fn() }));

import { resolveExportDelivery } from "./useExportDelivery";

const url = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents";

describe("resolveExportDelivery", () => {
  test("defaults to SharePoint and keeps local download as the alternative", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        disableExport: false,
      })
    ).toEqual({
      deliveries: [ExportDelivery.SHAREPOINT, ExportDelivery.LOCAL],
      sharepointUrl: url,
    });
  });

  test("offers only SharePoint when disableExport is on", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        disableExport: true,
      })
    ).toEqual({ deliveries: [ExportDelivery.SHAREPOINT], sharepointUrl: url });
  });

  test("offers only local download without a destination or the feature", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(resolveExportDelivery({ ...input, disableExport: false })).toEqual(
        { deliveries: [ExportDelivery.LOCAL] }
      );
    }
  });

  test("offers nothing when disableExport is on and SharePoint is not offered", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(resolveExportDelivery({ ...input, disableExport: true })).toEqual({
        deliveries: [],
      });
    }
  });
});
