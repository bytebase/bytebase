import dayjs from "dayjs";
import { describe, expect, test } from "vitest";
import {
  formatRangeScope,
  isWholeDays,
  matchQuickRange,
  parseRangeScope,
  parseTimeOfDay,
  quickRange,
  rangeEnd,
  rangeStart,
} from "./time-range";

const NOW = new Date(2026, 10, 24, 10, 42);
const local = (
  year: number,
  month: number,
  day: number,
  hour = 0,
  minute = 0,
  second = 0,
  ms = 0
) => new Date(year, month - 1, day, hour, minute, second, ms).getTime();

describe("quickRange", () => {
  test("covers whole days through the end of today", () => {
    expect(quickRange("today", NOW)).toEqual({
      from: local(2026, 11, 24),
      to: local(2026, 11, 24, 23, 59, 59, 999),
    });
    expect(quickRange("last-7-days", NOW)).toEqual({
      from: local(2026, 11, 18),
      to: local(2026, 11, 24, 23, 59, 59, 999),
    });
    expect(quickRange("last-30-days", NOW).from).toBe(local(2026, 10, 26));
  });

  test("month ranges start the day after the same date N months back", () => {
    expect(quickRange("last-3-months", NOW)).toEqual({
      from: local(2026, 8, 25),
      to: local(2026, 11, 24, 23, 59, 59, 999),
    });
    expect(quickRange("last-6-months", NOW).from).toBe(local(2026, 5, 25));
    expect(quickRange("last-12-months", NOW).from).toBe(local(2025, 11, 25));
  });

  test("a month range from a day the earlier month lacks stays in range", () => {
    // May 31 minus three months is the end of February.
    expect(quickRange("last-3-months", new Date(2026, 4, 31)).from).toBe(
      local(2026, 3, 1)
    );
  });
});

describe("matchQuickRange", () => {
  test("names a range only while it still means that quick range", () => {
    const lastWeek = quickRange("last-7-days", NOW);
    expect(matchQuickRange(lastWeek, NOW)).toBe("last-7-days");
    expect(
      matchQuickRange(lastWeek, dayjs(NOW).add(1, "day").toDate())
    ).toBeUndefined();
  });

  test("does not name a range with a different time of day", () => {
    const today = quickRange("today", NOW);
    expect(
      matchQuickRange({ ...today, to: local(2026, 11, 24, 23, 59) }, NOW)
    ).toBeUndefined();
  });
});

describe("parseTimeOfDay", () => {
  test.each([
    ["9", "09:00"],
    ["09", "09:00"],
    ["930", "09:30"],
    ["0930", "09:30"],
    ["9:5", "09:05"],
    ["23.59", "23:59"],
    [" 18:00 ", "18:00"],
  ])("reads %s as %s", (text, expected) => {
    expect(parseTimeOfDay(text)).toBe(expected);
  });

  test.each(["", "24:00", "12:60", "9am", "abc", "12345"])(
    "rejects %s",
    (text) => {
      expect(parseTimeOfDay(text)).toBeUndefined();
    }
  );
});

describe("range bounds", () => {
  const day = new Date(2026, 10, 20, 15, 30);

  test("the start is the first millisecond of its minute", () => {
    expect(rangeStart(day, "00:00")).toBe(local(2026, 11, 20));
    expect(rangeStart(day, "09:30")).toBe(local(2026, 11, 20, 9, 30));
  });

  test("an end at 23:59 includes the rest of the day", () => {
    expect(rangeEnd(day, "23:59")).toBe(local(2026, 11, 20, 23, 59, 59, 999));
    expect(rangeEnd(day, "18:00")).toBe(local(2026, 11, 20, 18));
  });

  test("whole days are 00:00 through 23:59", () => {
    expect(
      isWholeDays({
        from: rangeStart(day, "00:00"),
        to: rangeEnd(day, "23:59"),
      })
    ).toBe(true);
    expect(
      isWholeDays({
        from: rangeStart(day, "00:00"),
        to: rangeEnd(day, "00:00"),
      })
    ).toBe(false);
  });
});

describe("range scope value", () => {
  test("round-trips", () => {
    expect(parseRangeScope(formatRangeScope({ from: 1000, to: 2000 }))).toEqual(
      { from: 1000, to: 2000 }
    );
  });

  test.each([undefined, "", "1000", "1000,2000,3000", "a,b"])(
    "rejects %s",
    (value) => {
      expect(parseRangeScope(value)).toBeUndefined();
    }
  );
});
