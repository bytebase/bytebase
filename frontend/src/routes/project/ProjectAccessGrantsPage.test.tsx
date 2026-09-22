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
  shownTimestampTruncates,
} from "@/test-utils/humanizeTs";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { getAccessGrantDisplayStatusText } from "@/utils/accessGrant";

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

  test("reads status, creator, databases, statement, expiry, creation, then actions", () => {
    expect(columns.map((column) => column.key)).toEqual([
      "status",
      "creator",
      "databases",
      "statement",
      "expiration",
      "created",
      "actions",
    ]);
  });

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

  test("floors both dates at the timestamp minimum", () => {
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

  test("puts each cell under its own column", () => {
    const columns = grantColumns((key) => key);
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    const nowMs = Date.now();
    const createdMs = nowMs - 60_000;
    const expiresMs = nowMs + 86_400_000;
    act(() =>
      root.render(
        <table>
          <tbody>
            <AccessGrantRow
              columns={columns}
              grant={create(AccessGrantSchema, {
                name: "projects/p1/accessGrants/1",
                creator: "users/alice@example.com",
                status: AccessGrant_Status.ACTIVE,
                query: "SELECT secret FROM vault",
                createTime: timestampFromMs(createdMs),
                expiration: {
                  case: "expireTime",
                  value: timestampFromMs(expiresMs),
                },
              })}
              canActivate={false}
              canRevoke={true}
              onActivate={() => {}}
              onRevoke={() => {}}
            />
          </tbody>
        </table>
      )
    );

    const cells = Array.from(container.querySelectorAll("td"));
    const cell = (key: string) =>
      cells[columns.findIndex((column) => column.key === key)];
    expect(cells).toHaveLength(columns.length);
    expect(cell("status").textContent).toBe(
      getAccessGrantDisplayStatusText("ACTIVE")
    );
    expect(cell("creator").textContent).toBe("alice@example.com");
    expect(cell("databases").textContent).toBe("-");
    expect(cell("statement").textContent).toBe("SELECT secret FROM vault");
    expect(shownTimestampInstants(cell("expiration"))).toEqual([
      String(expiresMs),
    ]);
    expect(shownTimestampInstants(cell("created"))).toEqual([
      String(createdMs),
    ]);
    expect(cell("actions").textContent).toBe("sql-editor.revoke-access");
  });

  test("dates a grant's creation and expiry in one form, each fitting its box", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    act(() =>
      root.render(
        <table>
          <tbody>
            <AccessGrantRow
              columns={grantColumns((key) => key)}
              grant={create(AccessGrantSchema, {
                name: "projects/p1/accessGrants/1",
                createTime: timestampFromMs(1_000),
                expiration: {
                  case: "expireTime",
                  value: timestampFromMs(2_000),
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
    expect(shownTimestampTruncates(container)).toEqual([true, true]);
  });
});
