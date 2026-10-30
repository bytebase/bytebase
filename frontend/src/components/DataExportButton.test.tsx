import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ExportDestination, ExportFormat } from "@/types/proto-es/v1/common_pb";
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

const renderButton = (destinations: ExportDestination[]) =>
  render(
    <DataExportButton
      viewMode="DRAWER"
      supportFormats={[ExportFormat.CSV]}
      supportPassword
      destinations={destinations}
      sharepointUrl={sharepointUrl}
      onExport={onExport}
    />
  );

const openDrawer = () =>
  fireEvent.click(screen.getByRole("button", { name: "common.export" }));

describe("DataExportButton destination", () => {
  beforeEach(() => {
    onExport.mockReset();
  });

  test("defaults to SharePoint and hides the zip password for it", () => {
    renderButton([ExportDestination.SHAREPOINT, ExportDestination.DOWNLOAD]);
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

  test("shows the exported folder instead of downloading", async () => {
    onExport.mockImplementation(({ options, resolve }) => {
      expect(options.destination).toBe(ExportDestination.SHAREPOINT);
      expect(options.password).toBe("");
      resolve({
        sharepoint: [{ database: "main", status: "EXPORTED", uri: folderUri }],
      });
    });
    renderButton([ExportDestination.SHAREPOINT]);
    openDrawer();

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    expect(
      screen.queryByLabelText("export-data.password-optional")
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

    await waitFor(() => {
      expect(
        screen.getByText("export-data.sharepoint.exported-title")
      ).toBeInTheDocument();
    });
    expect(screen.getByText(decodeURI(folderUri))).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute("href", folderUri);
  });

  test("offers download alone with the zip password", () => {
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();

    expect(screen.queryAllByRole("radio")).toHaveLength(0);
    expect(
      screen.queryByText("export-data.destination.self")
    ).not.toBeInTheDocument();
    expect(
      screen.getByLabelText("export-data.password-optional")
    ).toBeInTheDocument();
  });

  test("disables export when no destination is available", () => {
    renderButton([]);
    expect(
      screen.getByRole("button", { name: "common.export" })
    ).toBeDisabled();
  });

  test("shows progress while the export runs", async () => {
    let finish: (() => void) | undefined;
    let reportProgress: DataExportRequest["onProgress"];
    onExport.mockImplementation(({ resolve, onProgress }) => {
      reportProgress = onProgress;
      finish = () => resolve({ sharepoint: [] });
    });
    renderButton([ExportDestination.SHAREPOINT]);
    openDrawer();
    fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

    expect(screen.getByRole("progressbar")).toHaveAccessibleName(
      "export-data.progress.sharepoint"
    );
    act(() => reportProgress?.(1, 3));
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "1"
    );
    expect(screen.getByRole("progressbar")).toHaveAccessibleName(
      "export-data.progress.databases"
    );

    await act(async () => finish?.());
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });
});
