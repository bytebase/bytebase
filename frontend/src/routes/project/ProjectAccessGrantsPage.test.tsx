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
  const preferred = columns.reduce((sum, c) => sum + c.defaultWidth, 0);
  const floors = columns.reduce(
    (sum, c) => sum + (c.resizable === false ? c.defaultWidth : c.minWidth),
    0
  );
  const full = (widths: number[], key: string) =>
    widths[at(key)] === columns[at(key)].defaultWidth;
  const floored = (widths: number[], key: string) =>
    widths[at(key)] === columns[at(key)].minWidth;

  test("opens both dates whole once the table holds every column", () => {
    for (const width of [preferred, preferred + 300]) {
      const widths = distributeColumnWidths(columns, width);
      expect(widths[at("created")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
      expect(widths[at("expiration")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
    }
  });

  test("gives way dates first, then statement, then creator, then databases", () => {
    for (let width = floors; width < preferred; width++) {
      const widths = distributeColumnWidths(columns, width);
      expect(widths.reduce((sum, w) => sum + w, 0)).toBe(width);
      columns.forEach((column, i) => {
        expect(widths[i]).toBeGreaterThanOrEqual(column.minWidth);
      });
      if (!full(widths, "statement")) {
        expect(floored(widths, "created") && floored(widths, "expiration")).toBe(
          true
        );
      }
      if (!full(widths, "creator")) {
        expect(floored(widths, "statement")).toBe(true);
      }
      if (!full(widths, "databases")) {
        expect(floored(widths, "creator")).toBe(true);
      }
      if (!full(widths, "status")) {
        expect(floored(widths, "databases")).toBe(true);
      }
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
