// @vitest-environment node
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import { getTimeForPbTimestampProtoEs } from "./timestamp";

describe("getTimeForPbTimestampProtoEs", () => {
  test("reads milliseconds, keeping the sub-millisecond nanos", () => {
    const timestamp = timestampFromMs(1_772_452_800_123);
    timestamp.nanos += 500_000;
    expect(getTimeForPbTimestampProtoEs(timestamp)).toBe(1_772_452_800_123.5);
  });

  test("uses the caller's fallback for an absent timestamp", () => {
    expect(getTimeForPbTimestampProtoEs(undefined, 0)).toBe(0);
  });

  test("refuses to invent a time when a caller bypasses the types", () => {
    // Substituting the current time would show "now" as when something
    // happened; the overloads make this unreachable from typed code.
    const untyped = getTimeForPbTimestampProtoEs as unknown as (
      timestamp?: undefined
    ) => number;
    expect(() => untyped(undefined)).toThrow();
  });
});
