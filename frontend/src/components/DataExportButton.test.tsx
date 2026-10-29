import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ExportDelivery, ExportFormat } from "@/types/proto-es/v1/common_pb";
import { type DataExportRequest, DataExportButton } from "./DataExportButton";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("./MaxRowCountSelect", () => ({
  MaxRowCountSelect: () => null,
}));

const sharepointUrl =
  "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents/Exports";
const folderUri = `${sharepointUrl}/20261022-main`;

const onExport = vi.fn<(request: DataExportRequest) => void>();

const renderButton = (deliveries: ExportDelivery[]) =>
  render(
    <DataExportButton
      viewMode="DRAWER"
      supportFormats={[ExportFormat.CSV]}
      supportPassword
      deliveries={deliveries}
      sharepointUrl={sharepointUrl}
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
    renderButton([ExportDelivery.SHAREPOINT, ExportDelivery.LOCAL]);
    openDrawer();

    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(2);
    expect(radios[0]).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText(decodeURI(sharepointUrl))).toBeInTheDocument();
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
      expect(options.delivery).toBe(ExportDelivery.SHAREPOINT);
      expect(options.password).toBe("");
      resolve({
        sharepoint: [{ database: "main", status: "DELIVERED", uri: folderUri }],
      });
    });
    renderButton([ExportDelivery.SHAREPOINT]);
    openDrawer();

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    expect(
      screen.queryByLabelText("export-data.password-optional")
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

    await waitFor(() => {
      expect(
        screen.getByText("export-data.sharepoint.delivered-title")
      ).toBeInTheDocument();
    });
    expect(screen.getByText(decodeURI(folderUri))).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute("href", folderUri);
  });

  test("offers local download alone with the zip password", () => {
    renderButton([ExportDelivery.LOCAL]);
    openDrawer();

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    expect(
      screen.queryByText("export-data.delivery.self")
    ).not.toBeInTheDocument();
    expect(
      screen.getByLabelText("export-data.password-optional")
    ).toBeInTheDocument();
  });

  test("disables export when no delivery is available", () => {
    renderButton([]);
    expect(
      screen.getByRole("button", { name: "common.export" })
    ).toBeDisabled();
  });
});
