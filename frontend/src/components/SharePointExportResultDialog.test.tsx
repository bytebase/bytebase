import { Code, ConnectError } from "@connectrpc/connect";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  SharePointExportResultDialog,
  toExportedSharePointResult,
  toFailedSharePointExport,
} from "./SharePointExportResultDialog";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("toFailedSharePointExport", () => {
  test("treats UNAVAILABLE as an upload that could not be confirmed", () => {
    expect(
      toFailedSharePointExport(
        "db",
        new ConnectError("files may exist", Code.Unavailable)
      )
    ).toEqual({ database: "db", status: "UNCONFIRMED", reason: "files may exist" });
  });

  test("treats other errors as failures", () => {
    expect(
      toFailedSharePointExport(
        "db",
        new ConnectError("not found", Code.FailedPrecondition)
      )
    ).toEqual({ database: "db", status: "FAILED", reason: "not found" });
    expect(toFailedSharePointExport("db", new Error("boom"))).toEqual({
      database: "db",
      status: "FAILED",
      reason: "Error: boom",
    });
  });
});

describe("toExportedSharePointResult", () => {
  test("uses the exported folder URI", () => {
    expect(
      toExportedSharePointResult("db", "https://folder", "https://destination")
    ).toEqual({
      database: "db",
      status: "EXPORTED",
      uri: "https://folder",
    });
  });

  test("links to the destination when the response has no URI", () => {
    expect(
      toExportedSharePointResult("db", "", "https://destination")
    ).toEqual({
      database: "db",
      status: "EXPORTED",
      uri: "https://destination",
    });
  });
});

describe("SharePointExportResultDialog", () => {
  test("shows exported, failed, and unconfirmed databases separately", () => {
    render(
      <SharePointExportResultDialog
        results={[
          {
            database: "a",
            status: "EXPORTED",
            uri: "https://contoso.sharepoint.com/Exports/20261022-a",
          },
          { database: "b", status: "FAILED", reason: "not found" },
          { database: "c", status: "UNCONFIRMED", reason: "files may exist" },
        ]}
        onClose={vi.fn()}
      />
    );

    expect(
      screen.getByText("export-data.sharepoint.result-title")
    ).toBeInTheDocument();
    expect(
      screen.getByText("https://contoso.sharepoint.com/Exports/20261022-a")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute(
      "href",
      "https://contoso.sharepoint.com/Exports/20261022-a"
    );
    expect(screen.getByText("not found")).toBeInTheDocument();
    expect(screen.getByText("files may exist")).toBeInTheDocument();
    expect(
      screen.getByText("export-data.sharepoint.status.unconfirmed")
    ).toBeInTheDocument();
  });

  test("titles an all-exported result as exported", () => {
    render(
      <SharePointExportResultDialog
        results={[
          { database: "a", status: "EXPORTED", uri: "https://a" },
        ]}
        onClose={vi.fn()}
      />
    );
    expect(
      screen.getByText("export-data.sharepoint.exported-title")
    ).toBeInTheDocument();
  });
});
