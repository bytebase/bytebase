import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import { SharePointLocationSchema } from "@/types/proto-es/v1/project_service_pb";
import { ExportRequest_Delivery } from "@/types/proto-es/v1/sql_service_pb";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/hooks/useAppState", () => ({ usePlanFeature: () => true }));
vi.mock("@/hooks/useProjectByName", () => ({ useProjectByName: vi.fn() }));
vi.mock("@/stores/app", () => ({ useAppStore: vi.fn() }));

import { resolveExportDelivery } from "./useExportDelivery";

const destination = create(SharePointLocationSchema, {
  siteName: "Finance",
  libraryName: "Documents",
});
const url = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents";

describe("resolveExportDelivery", () => {
  test("defaults to SharePoint and keeps local download as the alternative", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        destination,
        localDownloadDisabled: false,
      })
    ).toEqual({
      deliveries: [
        ExportRequest_Delivery.SHAREPOINT,
        ExportRequest_Delivery.LOCAL,
      ],
      destination,
    });
  });

  test("offers only SharePoint when local download is disabled", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: url,
        destination,
        localDownloadDisabled: true,
      }).deliveries
    ).toEqual([ExportRequest_Delivery.SHAREPOINT]);
  });

  test("offers only local download without a destination or the feature", () => {
    for (const input of [
      { hasSharePointFeature: true, sharepointUrl: "" },
      { hasSharePointFeature: false, sharepointUrl: url },
    ]) {
      expect(
        resolveExportDelivery({
          ...input,
          destination,
          localDownloadDisabled: false,
        })
      ).toEqual({ deliveries: [ExportRequest_Delivery.LOCAL] });
    }
  });

  test("explains why export is unavailable when no delivery remains", () => {
    expect(
      resolveExportDelivery({
        hasSharePointFeature: true,
        sharepointUrl: "",
        localDownloadDisabled: true,
      })
    ).toEqual({ deliveries: [], unavailableReason: "NO_DESTINATION" });
    expect(
      resolveExportDelivery({
        hasSharePointFeature: false,
        sharepointUrl: url,
        localDownloadDisabled: true,
      })
    ).toEqual({ deliveries: [], unavailableReason: "FEATURE_UNAVAILABLE" });
  });
});
