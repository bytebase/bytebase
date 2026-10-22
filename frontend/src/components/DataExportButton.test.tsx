import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { SharePointLocationSchema } from "@/types/proto-es/v1/project_service_pb";
import { ExportRequest_Delivery } from "@/types/proto-es/v1/sql_service_pb";
import { type DataExportRequest, DataExportButton } from "./DataExportButton";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("./MaxRowCountSelect", () => ({
  MaxRowCountSelect: () => null,
}));

const destination = create(SharePointLocationSchema, {
  siteName: "Finance",
  libraryName: "Documents",
  folderPath: "Exports",
});

const onExport = vi.fn<(request: DataExportRequest) => void>();

const renderButton = (deliveries: ExportRequest_Delivery[]) =>
  render(
    <DataExportButton
      viewMode="DRAWER"
      supportFormats={[ExportFormat.CSV]}
      supportPassword
      deliveries={deliveries}
      sharepointDestination={destination}
      unavailableMessage="export unavailable"
      onExport={onExport}
    />
  );

const openDrawer = () =>
  fireEvent.click(screen.getByRole("button", { name: "common.export" }));

describe("DataExportButton delivery", () => {
  beforeEach(() => {
    onExport.mockReset();
  });

  test("defaults to SharePoint and hides the zip password for it", () => {
    renderButton([ExportRequest_Delivery.SHAREPOINT, ExportRequest_Delivery.LOCAL]);
    openDrawer();

    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(2);
    expect(radios[0]).toHaveAttribute("aria-checked", "true");
    expect(
      screen.getByText("Finance / Documents / Exports")
    ).toBeInTheDocument();
    expect(
      screen.queryByLabelText("export-data.password-optional")
    ).not.toBeInTheDocument();

    fireEvent.click(radios[1]);
    expect(
      screen.getByLabelText("export-data.password-optional")
    ).toBeInTheDocument();
  });

  test("shows the delivered folder instead of downloading", async () => {
    onExport.mockImplementation(({ options, resolve }) => {
      expect(options.delivery).toBe(ExportRequest_Delivery.SHAREPOINT);
      expect(options.password).toBe("");
      resolve({
        sharepoint: [
          {
            database: "main",
            status: "DELIVERED",
            folderName: "20261022-main",
            webUrl: "https://folder",
          },
        ],
      });
    });
    renderButton([ExportRequest_Delivery.SHAREPOINT]);
    openDrawer();

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

    await waitFor(() => {
      expect(
        screen.getByText("export-data.sharepoint.delivered-title")
      ).toBeInTheDocument();
    });
    expect(screen.getByText("20261022-main")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute("href", "https://folder");
  });

  test("disables export when no delivery is available", () => {
    renderButton([]);
    expect(
      screen.getByRole("button", { name: "common.export" })
    ).toBeDisabled();
  });
});
