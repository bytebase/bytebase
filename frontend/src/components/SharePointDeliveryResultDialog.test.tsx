import { Code, ConnectError } from "@connectrpc/connect";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  SharePointDeliveryResultDialog,
  toDeliveredSharePointResult,
  toFailedSharePointDelivery,
} from "./SharePointDeliveryResultDialog";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("toFailedSharePointDelivery", () => {
  test("treats UNAVAILABLE as a delivery that could not be confirmed", () => {
    expect(
      toFailedSharePointDelivery(
        "db",
        new ConnectError("files may exist", Code.Unavailable)
      )
    ).toEqual({ database: "db", status: "UNCONFIRMED", reason: "files may exist" });
  });

  test("treats other errors as failures", () => {
    expect(
      toFailedSharePointDelivery(
        "db",
        new ConnectError("not found", Code.FailedPrecondition)
      )
    ).toEqual({ database: "db", status: "FAILED", reason: "not found" });
    expect(toFailedSharePointDelivery("db", new Error("boom"))).toEqual({
      database: "db",
      status: "FAILED",
      reason: "Error: boom",
    });
  });
});

describe("toDeliveredSharePointResult", () => {
  test("links to the destination when the response has no delivery", () => {
    expect(
      toDeliveredSharePointResult("db", undefined, "https://destination")
    ).toEqual({
      database: "db",
      status: "DELIVERED",
      folderName: "",
      webUrl: "https://destination",
    });
  });
});

describe("SharePointDeliveryResultDialog", () => {
  test("shows delivered, failed, and unconfirmed databases separately", () => {
    render(
      <SharePointDeliveryResultDialog
        results={[
          {
            database: "a",
            status: "DELIVERED",
            folderName: "20261022-a",
            webUrl: "https://a",
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
    expect(screen.getByText("20261022-a")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "export-data.sharepoint.open" })
    ).toHaveAttribute("href", "https://a");
    expect(screen.getByText("not found")).toBeInTheDocument();
    expect(screen.getByText("files may exist")).toBeInTheDocument();
    expect(
      screen.getByText("export-data.sharepoint.status.unconfirmed")
    ).toBeInTheDocument();
  });

  test("titles an all-delivered result as delivered", () => {
    render(
      <SharePointDeliveryResultDialog
        results={[
          {
            database: "a",
            status: "DELIVERED",
            folderName: "f",
            webUrl: "https://a",
          },
        ]}
        onClose={vi.fn()}
      />
    );
    expect(
      screen.getByText("export-data.sharepoint.delivered-title")
    ).toBeInTheDocument();
  });
});
