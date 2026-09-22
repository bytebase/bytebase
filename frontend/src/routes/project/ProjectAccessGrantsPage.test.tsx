import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  TIMESTAMP_COLUMN_MIN_WIDTH,
  TIMESTAMP_COLUMN_WIDTH,
} from "@/components/timestampColumn";
import { distributeColumnWidths } from "@/hooks/useColumnWidths";
import {
  shownTimestampInstants,
  shownTimestampModes,
} from "@/test-utils/humanizeTs";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

import { AccessGrantRow, grantColumns } from "./ProjectAccessGrantsPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("grantColumns", () => {
  const columns = grantColumns((key) => key);
  const at = (key: string) => columns.findIndex((column) => column.key === key);

  test.each([1100, 1300, 1500, 1800])(
    "opens both dates whole in a %ipx table",
    (containerWidth) => {
      const widths = distributeColumnWidths(columns, containerWidth);
      expect(widths[at("created")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
      expect(widths[at("expiration")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
    }
  );

  test("fits the container exactly at every width that holds its floors", () => {
    const floors = columns.reduce(
      (sum, column) =>
        sum +
        (column.resizable === false || column.grow === false
          ? column.defaultWidth
          : column.minWidth),
      0
    );
    for (let containerWidth = floors; containerWidth <= 1800; containerWidth++) {
      const widths = distributeColumnWidths(columns, containerWidth);
      expect(widths.reduce((sum, w) => sum + w, 0)).toBe(containerWidth);
      columns.forEach((column, i) => {
        expect(widths[i]).toBeGreaterThanOrEqual(column.minWidth);
      });
      expect(widths[at("created")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
      expect(widths[at("expiration")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
    }
  });

  test("lets a reader narrow either date to the date alone", () => {
    expect(columns[at("created")].minWidth).toBe(TIMESTAMP_COLUMN_MIN_WIDTH);
    expect(columns[at("expiration")].minWidth).toBe(TIMESTAMP_COLUMN_MIN_WIDTH);
  });
});

describe("AccessGrantRow", () => {
  const roots: ReturnType<typeof createRoot>[] = [];
  afterEach(() => {
    for (const root of roots.splice(0)) {
      act(() => root.unmount());
    }
  });

  test("dates a grant's creation and expiry in one form, each ellipsizing", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    const nowMs = Date.now();
    act(() =>
      root.render(
        <table>
          <tbody>
            <AccessGrantRow
              grant={create(AccessGrantSchema, {
                name: "projects/p1/accessGrants/1",
                creator: "users/alice@example.com",
                status: AccessGrant_Status.ACTIVE,
                createTime: timestampFromMs(nowMs - 60_000),
                expiration: {
                  case: "expireTime",
                  value: timestampFromMs(nowMs + 86_400_000),
                },
              })}
              canActivate={false}
              canRevoke={false}
              onActivate={() => {}}
              onRevoke={() => {}}
            />
          </tbody>
        </table>
      )
    );

    expect(shownTimestampModes(container)).toEqual([
      "operational",
      "operational",
    ]);
    expect(shownTimestampInstants(container)).toEqual([
      String(nowMs - 60_000),
      String(nowMs + 86_400_000),
    ]);
    for (const date of container.querySelectorAll("[data-testid=humanize-ts]")) {
      expect(date.className).toContain("truncate");
    }
  });
});
