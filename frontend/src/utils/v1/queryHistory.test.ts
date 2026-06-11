import { describe, expect, test } from "vitest";
import { extractQueryHistoryUID } from "./queryHistory";

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
