import { create } from "@bufbuild/protobuf";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  type Release_File,
  Release_FileSchema,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

let ReleaseFileTable: typeof import("./ReleaseFileTable").ReleaseFileTable;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const makeFile = (version: string): Release_File =>
  create(Release_FileSchema, {
    version,
    path: `migrations/${version}.sql`,
    sheet: `sheets/${version}`,
    sheetSha256: "abcdef1234567890",
  });

describe("ReleaseFileTable", () => {
  test("renders version/type/filename cells for each file", async () => {
    ({ ReleaseFileTable } = await import("./ReleaseFileTable"));
    const files = [makeFile("v1"), makeFile("v2")];
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseFileTable, {
        files,
        releaseType: Release_Type.VERSIONED,
      })
    );

    const rows = container.querySelectorAll("tbody tr");
    expect(rows.length).toBe(2);
    expect(container.textContent).toContain("v1");
    expect(container.textContent).toContain("v2");
    expect(container.textContent).toContain("migrations/v1.sql");
    // VERSIONED → i18n key returned by the mocked translator
    expect(container.textContent).toContain("issue.title.change-database");
    unmount();
  });

  test("hides selection column by default, shows it when enabled", async () => {
    ({ ReleaseFileTable } = await import("./ReleaseFileTable"));
    const files = [makeFile("v1")];

    const hidden = renderIntoContainer(
      createElement(ReleaseFileTable, {
        files,
        releaseType: Release_Type.DECLARATIVE,
      })
    );
    expect(
      hidden.container.querySelectorAll('input[type="checkbox"]').length
    ).toBe(0);
    hidden.unmount();

    const shown = renderIntoContainer(
      createElement(ReleaseFileTable, {
        files,
        releaseType: Release_Type.DECLARATIVE,
        showSelection: true,
      })
    );
    expect(
      shown.container.querySelectorAll('input[type="checkbox"]').length
    ).toBeGreaterThan(0);
    shown.unmount();
  });

  test("fires onRowClick with the clicked file when rowClickable", async () => {
    ({ ReleaseFileTable } = await import("./ReleaseFileTable"));
    const files = [makeFile("v1"), makeFile("v2")];
    const onRowClick = vi.fn();
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseFileTable, {
        files,
        releaseType: Release_Type.VERSIONED,
        onRowClick,
      })
    );

    const rows = container.querySelectorAll("tbody tr");
    act(() => {
      rows[1]?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(onRowClick).toHaveBeenCalledTimes(1);
    expect(onRowClick.mock.calls[0][0]).toBe(files[1]);
    unmount();
  });
});
