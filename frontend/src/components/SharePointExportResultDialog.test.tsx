import { ErrorInfoSchema } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  mergeSharePointResults,
  type SharePointExportResult,
  SharePointExportResultDialog,
  toExportedSharePointResult,
  toFailedSharePointExport,
} from "./SharePointExportResultDialog";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, string>) =>
      options?.folder ? `${key} ${options.folder}` : key,
  }),
}));

const db = { name: "instances/prod/databases/db", label: "Prod/db" };
const folder = "20261022T010507Z-alice@example.com-db-0a1b2c3d";

const sharePointError = (
  message: string,
  code: Code,
  reason: string,
  metadata: Record<string, string> = {}
) =>
  new ConnectError(message, code, undefined, [
    {
      desc: ErrorInfoSchema,
      value: { reason, domain: "bytebase.com", metadata },
    },
  ]);

const failed = (
  name: string,
  status: "FAILED" | "UNCONFIRMED",
  failure: { reason?: string; folder?: string; message: string }
): SharePointExportResult => ({ name, database: name, status, failure });

const exported = (name: string): SharePointExportResult => ({
  name,
  database: name,
  status: "EXPORTED",
  uri: `https://contoso.sharepoint.com/Exports/${name}`,
});

describe("toFailedSharePointExport", () => {
  test("treats UNAVAILABLE as an upload that could not be confirmed", () => {
    expect(
      toFailedSharePointExport(
        db,
        sharePointError(
          "could not be confirmed",
          Code.Unavailable,
          "SHAREPOINT_UPLOAD_UNCONFIRMED",
          { folder }
        )
      )
    ).toEqual({
      name: db.name,
      database: db.label,
      status: "UNCONFIRMED",
      failure: {
        reason: "SHAREPOINT_UPLOAD_UNCONFIRMED",
        folder,
        message: "could not be confirmed",
      },
    });
  });

  test("treats other errors as failures", () => {
    expect(
      toFailedSharePointExport(
        db,
        sharePointError(
          "not found",
          Code.FailedPrecondition,
          "SHAREPOINT_DESTINATION_NOT_FOUND"
        )
      )
    ).toEqual({
      name: db.name,
      database: db.label,
      status: "FAILED",
      failure: {
        reason: "SHAREPOINT_DESTINATION_NOT_FOUND",
        message: "not found",
      },
    });
    expect(
      toFailedSharePointExport(
        db,
        new ConnectError("syntax error", Code.InvalidArgument)
      )
    ).toEqual({
      name: db.name,
      database: db.label,
      status: "FAILED",
      failure: { message: "syntax error" },
    });
    expect(toFailedSharePointExport(db, new Error("boom"))).toEqual({
      name: db.name,
      database: db.label,
      status: "FAILED",
      failure: { message: "Error: boom" },
    });
  });

  test("treats an aborted request as canceled in flight", () => {
    expect(
      toFailedSharePointExport(
        db,
        new ConnectError("The operation was aborted.", Code.Canceled)
      )
    ).toEqual({ name: db.name, database: db.label, status: "CANCELED" });
  });
});

describe("toExportedSharePointResult", () => {
  test("uses the exported folder URI", () => {
    expect(
      toExportedSharePointResult(db, "https://folder", "https://destination")
    ).toEqual({
      name: db.name,
      database: db.label,
      status: "EXPORTED",
      uri: "https://folder",
    });
  });

  test("links to the destination when the response has no URI", () => {
    expect(toExportedSharePointResult(db, "", "https://destination")).toEqual({
      name: db.name,
      database: db.label,
      status: "EXPORTED",
      uri: "https://destination",
    });
  });
});

describe("mergeSharePointResults", () => {
  test("replaces retried rows in place", () => {
    const a = failed("a", "FAILED", { message: "boom" });
    const b = exported("b");
    const c = failed("c", "UNCONFIRMED", { message: "unknown" });
    expect(
      mergeSharePointResults([a, b, c], [exported("c"), exported("a")])
    ).toEqual([exported("a"), b, exported("c")]);
  });
});

describe("SharePointExportResultDialog", () => {
  test("shows exported, failed, and unconfirmed databases separately", () => {
    render(
      <SharePointExportResultDialog
        results={[
          exported("a"),
          failed("b", "FAILED", { message: "not found" }),
          failed("c", "UNCONFIRMED", { message: "files may exist" }),
        ]}
        onClose={vi.fn()}
      />
    );

    expect(
      screen.getByText("export-data.sharepoint.result-title")
    ).toBeInTheDocument();
    expect(
      screen.getByText("https://contoso.sharepoint.com/Exports/a")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute("href", "https://contoso.sharepoint.com/Exports/a");
    expect(screen.getByText("not found")).toBeInTheDocument();
    expect(screen.getByText("files may exist")).toBeInTheDocument();
    expect(
      screen.getByText("export-data.sharepoint.status.unconfirmed")
    ).toBeInTheDocument();
  });

  test("explains a SharePoint failure and keeps the server message in details", () => {
    const message =
      'the upload to SharePoint failed: Microsoft Graph returned HTTP 507 quotaLimitReached';
    render(
      <SharePointExportResultDialog
        results={[
          failed("a", "FAILED", {
            reason: "SHAREPOINT_STORAGE_FULL",
            folder,
            message,
          }),
        ]}
        onClose={vi.fn()}
      />
    );

    const row = screen.getByRole("listitem");
    expect(row).toHaveTextContent("export-data.sharepoint.error.storage-full");
    expect(row).toHaveTextContent(
      `export-data.sharepoint.error.files-may-exist ${folder}`
    );
    expect(screen.queryByText(message)).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "error-page.error-details" })
    );
    expect(screen.getByText(message)).toBeInTheDocument();
  });

  test("titles an all-exported result as exported", () => {
    render(
      <SharePointExportResultDialog
        results={[exported("a")]}
        onRetry={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(
      screen.getByText("export-data.sharepoint.exported-title")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", {
        name: "export-data.sharepoint.retry-failed",
      })
    ).not.toBeInTheDocument();
  });

  test("retries only failed and unconfirmed databases", () => {
    const onRetry = vi.fn();
    render(
      <SharePointExportResultDialog
        results={[
          exported("a"),
          failed("b", "FAILED", { message: "boom" }),
          { name: "c", database: "c", status: "CANCELED" },
          failed("d", "UNCONFIRMED", { message: "unknown" }),
        ]}
        onRetry={onRetry}
        onClose={vi.fn()}
      />
    );

    expect(
      screen.getByText("export-data.sharepoint.retry-unconfirmed-warning")
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", {
        name: "export-data.sharepoint.retry-failed",
      })
    );
    expect(onRetry).toHaveBeenCalledWith(["b", "d"]);
  });

  test("warns about duplicates only when an unconfirmed database is retried", () => {
    render(
      <SharePointExportResultDialog
        results={[failed("a", "FAILED", { message: "boom" })]}
        onRetry={vi.fn()}
        onClose={vi.fn()}
      />
    );
    expect(
      screen.queryByText("export-data.sharepoint.retry-unconfirmed-warning")
    ).not.toBeInTheDocument();
  });

  test("shows a running retry and cancels it", () => {
    const onCancelRetry = vi.fn();
    render(
      <SharePointExportResultDialog
        results={[exported("a"), failed("b", "FAILED", { message: "boom" })]}
        retry={{ names: new Set(["b"]), elapsedSeconds: 12 }}
        onRetry={vi.fn()}
        onCancelRetry={onCancelRetry}
        onClose={vi.fn()}
      />
    );

    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("export-data.sharepoint.status.exported");
    expect(rows[1]).toHaveTextContent("export-data.sharepoint.status.retrying");
    expect(
      within(screen.getByRole("progressbar")).getByText("12s")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "common.close" })
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(onCancelRetry).toHaveBeenCalled();
  });

  test("says files may exist for a database canceled in flight", () => {
    render(
      <SharePointExportResultDialog
        results={[
          exported("a"),
          { name: "b", database: "b", status: "CANCELED" },
          { name: "c", database: "c", status: "NOT_STARTED" },
        ]}
        onClose={vi.fn()}
      />
    );

    expect(
      screen.getByText("export-data.sharepoint.result-title")
    ).toBeInTheDocument();
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("export-data.sharepoint.status.exported");
    expect(rows[1]).toHaveTextContent("export-data.sharepoint.status.canceled");
    expect(rows[1]).toHaveTextContent(
      "export-data.sharepoint.canceled-description"
    );
    expect(rows[2]).toHaveTextContent(
      "export-data.sharepoint.status.not-started"
    );
    expect(rows[2]).toHaveTextContent(
      "export-data.sharepoint.not-started-description"
    );
  });
});
