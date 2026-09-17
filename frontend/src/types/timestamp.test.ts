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

  test("has no reading for an absent timestamp without a fallback", () => {
    // The current time or the epoch would each show a plausible-looking time.
    const absent:
      | Parameters<typeof getTimeForPbTimestampProtoEs>[0]
      | undefined = undefined;
    expect(getTimeForPbTimestampProtoEs(absent)).toBeUndefined();
  });
});
