import { Code, ConnectError } from "@connectrpc/connect";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { pushNotification } from "@/stores";
import { ExportDestination, ExportFormat } from "@/types/proto-es/v1/common_pb";
import { type DataExportRequest, DataExportButton } from "./DataExportButton";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("./MaxRowCountSelect", () => ({
  MaxRowCountSelect: () => null,
}));

// Zipping a multi-database download waits on the test, so it can cancel
// while the zip is being generated.
const zipMocks = vi.hoisted(() => ({
  finishZip: undefined as (() => void) | undefined,
  saveAs: vi.fn(),
}));

vi.mock("jszip", () => ({
  default: class {
    file() {}
    generateAsync() {
      return new Promise<Blob>((resolve) => {
        zipMocks.finishZip = () => resolve(new Blob());
      });
    }
  },
}));

vi.mock("file-saver", () => ({ default: zipMocks.saveAs }));

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

const confirmExport = () =>
  fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

const cancel = () =>
  fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

describe("DataExportButton destination", () => {
  beforeEach(() => {
    onExport.mockReset();
    vi.mocked(pushNotification).mockReset();
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
      /^export-data\.progress\.sharepoint \d+s$/
    );
    act(() => reportProgress?.(1, 3));
    expect(screen.getByRole("progressbar")).toHaveAttribute(
      "aria-valuenow",
      "1"
    );
    expect(screen.getByRole("progressbar")).toHaveAccessibleName(
      /^export-data\.progress\.databases \d+s$/
    );

    await act(async () => finish?.());
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  test("explains a long SharePoint export with the elapsed time", async () => {
    vi.useFakeTimers();
    try {
      let finish: (() => void) | undefined;
      onExport.mockImplementation(({ resolve }) => {
        finish = () => resolve({ sharepoint: [] });
      });
      renderButton([ExportDestination.SHAREPOINT]);
      openDrawer();
      fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

      expect(screen.getByText("0s")).toBeInTheDocument();
      act(() => vi.advanceTimersByTime(10_000));
      expect(screen.getByText("10s")).toBeInTheDocument();
      expect(
        screen.queryByText("export-data.progress.sharepoint-slow")
      ).not.toBeInTheDocument();

      act(() => vi.advanceTimersByTime(65_000));
      expect(screen.getByText("75s")).toBeInTheDocument();
      expect(
        screen.getByText("export-data.progress.sharepoint-slow")
      ).toBeInTheDocument();

      await act(async () => finish?.());
      expect(
        screen.queryByText("export-data.progress.sharepoint-slow")
      ).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  test("keeps a long download's progress unchanged", () => {
    vi.useFakeTimers();
    try {
      onExport.mockImplementation(() => {});
      renderButton([ExportDestination.DOWNLOAD]);
      openDrawer();
      fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));

      act(() => vi.advanceTimersByTime(75_000));
      expect(screen.getByRole("progressbar")).toHaveAccessibleName(
        "export-data.progress.download"
      );
      expect(
        screen.queryByText("export-data.progress.sharepoint-slow")
      ).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("DataExportButton cancel", () => {
  beforeEach(() => {
    onExport.mockReset();
    vi.mocked(pushNotification).mockReset();
    window.URL.createObjectURL = vi.fn(() => "blob:export");
    window.URL.revokeObjectURL = vi.fn();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("aborts a running download and reports nothing", async () => {
    let signal: AbortSignal | undefined;
    onExport.mockImplementation((request) => {
      signal = request.signal;
      request.signal.addEventListener("abort", () =>
        request.reject(new ConnectError("aborted", Code.Canceled))
      );
    });
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();
    confirmExport();
    expect(signal?.aborted).toBe(false);

    cancel();

    expect(signal?.aborted).toBe(true);
    await waitFor(() => {
      expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    });
    expect(pushNotification).not.toHaveBeenCalled();
  });

  test("saves nothing from a download canceled as it finished", async () => {
    onExport.mockImplementation((request) => {
      request.signal.addEventListener("abort", () =>
        request.resolve([{ content: new Uint8Array([1]), filename: "a.zip" }])
      );
    });
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();
    confirmExport();
    cancel();

    await waitFor(() => {
      expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    });
    expect(window.URL.createObjectURL).not.toHaveBeenCalled();
    expect(pushNotification).not.toHaveBeenCalled();
  });

  test("saves nothing from a multi-database download canceled while zipping", async () => {
    zipMocks.finishZip = undefined;
    zipMocks.saveAs.mockReset();
    onExport.mockImplementation(({ resolve }) =>
      resolve([
        { content: new Uint8Array([1]), filename: "a.zip" },
        { content: new Uint8Array([2]), filename: "b.zip" },
      ])
    );
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();
    confirmExport();
    await waitFor(() => expect(zipMocks.finishZip).toBeDefined());

    cancel();
    await act(async () => zipMocks.finishZip?.());

    await waitFor(() => {
      expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
    });
    expect(zipMocks.saveAs).not.toHaveBeenCalled();
    expect(pushNotification).not.toHaveBeenCalled();
  });

  test("shows what a canceled SharePoint export left behind", async () => {
    onExport.mockImplementation((request) => {
      request.signal.addEventListener("abort", () =>
        request.resolve({
          sharepoint: [{ database: "main", status: "CANCELED" }],
        })
      );
    });
    renderButton([ExportDestination.SHAREPOINT]);
    openDrawer();
    confirmExport();
    cancel();

    await waitFor(() => {
      expect(
        screen.getByText("export-data.sharepoint.canceled-description")
      ).toBeInTheDocument();
    });
    expect(pushNotification).not.toHaveBeenCalled();
  });

  const pressEscape = () =>
    fireEvent.keyDown(document.activeElement ?? document.body, {
      key: "Escape",
    });

  const pressOutside = () => {
    fireEvent.pointerDown(document.body);
    fireEvent.mouseDown(document.body);
    fireEvent.pointerUp(document.body);
    fireEvent.mouseUp(document.body);
    fireEvent.click(document.body);
  };

  const startPendingExport = () => {
    let signal: AbortSignal | undefined;
    onExport.mockImplementation((request) => {
      signal = request.signal;
    });
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();
    confirmExport();
    return () => signal;
  };

  test("Escape and an outside click close the drawer when no export runs", async () => {
    renderButton([ExportDestination.DOWNLOAD]);
    openDrawer();
    pressEscape();
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    openDrawer();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    pressOutside();
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  test("Escape and an outside click keep a running export open and running", () => {
    const signal = startPendingExport();

    pressEscape();
    pressOutside();

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    expect(signal()?.aborted).toBe(false);
  });

  test("the close button aborts a running export", async () => {
    const signal = startPendingExport();

    fireEvent.click(screen.getByRole("button", { name: "common.close" }));

    expect(signal()?.aborted).toBe(true);
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
  });

  test("the password dialog stays open on Escape and an outside click while exporting", async () => {
    let signal: AbortSignal | undefined;
    onExport.mockImplementation((request) => {
      signal = request.signal;
    });
    render(
      <DataExportButton
        viewMode="DROPDOWN"
        supportFormats={[ExportFormat.CSV]}
        supportPassword
        onExport={onExport}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "common.export" }));
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "sql-editor.download-as-file" })
    );
    fireEvent.click(await screen.findByRole("button", { name: "common.export" }));

    pressEscape();
    pressOutside();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(signal?.aborted).toBe(false);

    cancel();
    expect(signal?.aborted).toBe(true);
  });
});
