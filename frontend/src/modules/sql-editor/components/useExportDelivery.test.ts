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

const both = [ExportDelivery.LOCAL, ExportDelivery.SHAREPOINT];

describe("resolveExportDelivery", () => {
  test("defaults to SharePoint and keeps local download as the alternative", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDeliveries: both,
      })
    ).toEqual({
      deliveries: [ExportDelivery.SHAREPOINT, ExportDelivery.LOCAL],
      sharepointUrl: url,
    });
  });

  test("offers only SharePoint when the policy allows only SharePoint", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDeliveries: [ExportDelivery.SHAREPOINT],
      })
    ).toEqual({ deliveries: [ExportDelivery.SHAREPOINT], sharepointUrl: url });
  });

  test("offers only local download when the policy allows only local download", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        allowedDeliveries: [ExportDelivery.LOCAL],
      })
    ).toEqual({ deliveries: [ExportDelivery.LOCAL] });
  });

  test("offers only local download without a destination or the feature", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDelivery({ ...input, allowedDeliveries: both })
      ).toEqual({ deliveries: [ExportDelivery.LOCAL] });
    }
  });

  test("offers nothing when the policy allows only SharePoint and it is not configured", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDelivery({
          ...input,
          allowedDeliveries: [ExportDelivery.SHAREPOINT],
        })
      ).toEqual({ deliveries: [] });
    }
  });
});
