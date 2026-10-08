// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { describe, expect, test, vi } from "vitest";
import { QueryHistorySchema } from "@/types/proto-es/v1/query_history_service_pb";
import { extractQueryHistoryUID, queryHistoryTabTitle } from "./queryHistory";

// The title is built from Intl, which reads the active locale here.
vi.mock("@/lib/i18n", () => ({ default: { language: "en-US" } }));

describe("extractQueryHistoryUID", () => {
  test("extracts the uid from a full resource name", () => {
    expect(
      extractQueryHistoryUID(
        "projects/proj1/queryHistories/550e8400-e29b-41d4-a716-446655440000"
      )
    ).toBe("550e8400-e29b-41d4-a716-446655440000");
  });

  test("returns the unknown id sentinel when no match", () => {
    expect(extractQueryHistoryUID("projects/proj1")).toBe("-1");
  });
});

describe("queryHistoryTabTitle", () => {
  test("names the time in full, since a tab title hosts no tooltip", () => {
    expect(
      queryHistoryTabTitle(
        create(QueryHistorySchema, {
          createTime: timestampFromMs(Date.UTC(2026, 2, 2, 12)),
        })
      )
    ).toBe("Query history at Mar 2, 2026, 8:00:00 PM GMT+8");
  });

  test("names no time when the history carries none", () => {
    expect(queryHistoryTabTitle(create(QueryHistorySchema, {}))).toBe(
      "Query history"
    );
  });
});
