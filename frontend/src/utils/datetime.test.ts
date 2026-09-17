// @vitest-environment node
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

// Absolute renderings below are asserted against literals from the design doc;
// the fixed Asia/Shanghai zone they assume is pinned in vitest.config.ts.
const i18n = vi.hoisted(() => ({ language: "en-US" }));

vi.mock("@/lib/i18n", () => ({
  default: i18n,
}));

const withLocale = (language: string, run: () => void) => {
  const previous = i18n.language;
  i18n.language = language;
  try {
    run();
  } finally {
    i18n.language = previous;
  }
};

import {
  formatAbsoluteDate,
  formatAbsoluteDateTime,
  formatCompactDateTime,
  formatOperationalDateTime,
  formatQueueTime,
  formatRelativeTime,
  hasPassed,
  nextAbsoluteDateChangeAt,
  nextCountdownChangeAt,
  nextDaysLeftChangeAt,
  nextPassedAt,
  nextQueueTimeChangeAt,
  nextRelativeTimeChangeAt,
  RELATIVE_THRESHOLD_MS,
  readCountdown,
  readDaysLeft,
} from "./datetime";

describe("formatRelativeTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("returns 'now' for timestamps less than 10 seconds ago", () => {
    const result = formatRelativeTime(Date.now() - 5000);
    expect(result).toBe("now");
  });

  test("returns 'X seconds ago' for 10-59 seconds", () => {
    const result = formatRelativeTime(Date.now() - 30_000);
    expect(result).toContain("seconds ago");
  });

  test("returns 'X minutes ago' for 1-59 minutes", () => {
    const result = formatRelativeTime(Date.now() - 5 * 60_000);
    expect(result).toMatch(/minutes? ago/);
  });

  test("returns 'X hours ago' for 1-23 hours", () => {
    const result = formatRelativeTime(Date.now() - 3 * 3_600_000);
    expect(result).toMatch(/hours? ago/);
  });

  test("returns 'yesterday' for ~24 hours ago", () => {
    const result = formatRelativeTime(Date.now() - 24 * 3_600_000);
    expect(result).toBe("yesterday");
  });

  test("returns 'X days ago' for 2-30 days", () => {
    const result = formatRelativeTime(Date.now() - 10 * 86_400_000);
    expect(result).toContain("days ago");
  });
});

describe("formatAbsoluteDateTime", () => {
  test("includes month, day, year, time, and seconds", () => {
    const ts = new Date("2026-03-02T14:30:00Z").getTime();
    const result = formatAbsoluteDateTime(ts);
    expect(result).toContain("Mar");
    expect(result).toContain("2026");
    expect(result).toContain("2");
    expect(result).toContain("00");
  });
});

describe("formatAbsoluteDate", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("omits year for same-year dates", () => {
    const ts = new Date("2026-06-15T00:00:00Z").getTime();
    const result = formatAbsoluteDate(ts);
    expect(result).not.toContain("2026");
    expect(result).toContain("Jun");
  });

  test("includes year for different-year dates", () => {
    const ts = new Date("2025-01-15T00:00:00Z").getTime();
    const result = formatAbsoluteDate(ts);
    expect(result).toContain("2025");
  });
});

describe("formatQueueTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("reads as relative age inside the 30-day window", () => {
    expect(formatQueueTime(Date.now() - 6 * 86_400_000)).toBe("6 days ago");
  });

  test("still reads as relative age one day short of the threshold", () => {
    expect(formatQueueTime(Date.now() - 29 * 86_400_000)).toBe("29 days ago");
  });

  test("switches to an absolute date at the threshold", () => {
    expect(formatQueueTime(Date.now() - RELATIVE_THRESHOLD_MS)).toBe("Jan 31");
  });

  test("keeps the year on a date from another year", () => {
    expect(formatQueueTime(Date.now() - 90 * 86_400_000)).toBe("Dec 2, 2025");
  });

  test("measures a future timestamp by the same threshold", () => {
    expect(formatQueueTime(Date.now() + 2 * 86_400_000)).toBe("in 2 days");
    expect(formatQueueTime(Date.now() + 40 * 86_400_000)).toBe("Apr 11");
  });

  test("renders both branches in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(formatQueueTime(Date.now() - 6 * 86_400_000)).toBe("6天前");
      expect(formatQueueTime(Date.now() - RELATIVE_THRESHOLD_MS)).toBe(
        "1月31日"
      );
    });
  });
});

describe("formatCompactDateTime", () => {
  const ts = new Date("2026-08-26T06:03:22Z").getTime();

  test("drops the seconds and the timezone a history row does not need", () => {
    expect(formatCompactDateTime(ts)).toBe("Aug 26, 2026, 2:03 PM");
  });

  test("renders in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(formatCompactDateTime(ts)).toBe("2026年8月26日 14:03");
    });
  });
});

describe("formatOperationalDateTime", () => {
  // The value a reader is about to act on: 9am Shanghai, with a sub-minute
  // tail of the kind a `now() + N days` expiration preset writes.
  const ts = new Date("2026-09-15T01:00:22Z").getTime();

  test("names the timezone in the visible string", () => {
    expect(formatOperationalDateTime(ts)).toBe("Sep 15, 2026, 9:00 AM GMT+8");
  });

  test("floors to the minute rather than showing the seconds", () => {
    expect(formatOperationalDateTime(ts)).toBe(
      formatOperationalDateTime(new Date("2026-09-15T01:00:00Z").getTime())
    );
  });

  test("renders in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(formatOperationalDateTime(ts)).toBe("2026年9月15日 GMT+8 09:00");
    });
  });
});

const SECOND_MS = 1_000;
const MINUTE_MS = 60_000;
const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;

// Ages around every edge the relative reading has: the "now" threshold, each
// unit switch, the half-unit points where a rounded count turns over, and the
// 30-day switch. Negative ages are future timestamps.
const SAMPLED_AGES_MS = [
  0,
  5 * SECOND_MS,
  9_999,
  10 * SECOND_MS,
  10_400,
  10_500,
  30 * SECOND_MS,
  59_400,
  59_500,
  MINUTE_MS,
  89 * SECOND_MS,
  90 * SECOND_MS,
  119 * SECOND_MS,
  30 * MINUTE_MS,
  59.5 * MINUTE_MS,
  HOUR_MS,
  1.5 * HOUR_MS,
  23.4 * HOUR_MS,
  DAY_MS,
  1.2 * DAY_MS,
  1.5 * DAY_MS,
  29 * DAY_MS,
  29.5 * DAY_MS,
  30 * DAY_MS - 1,
  30 * DAY_MS,
  45 * DAY_MS,
  400 * DAY_MS,
].flatMap((ageMs) => (ageMs === 0 ? [0] : [ageMs, -ageMs]));

// The contract a boundary keeps with the reading it schedules: the reading
// holds right up to the named instant and changes at it, and the boundary names
// that same instant from anywhere before it. A boundary checked on its own
// cannot see that it was derived from the wrong reading.
const expectBoundaryMatchesReading = (
  read: (tsMs: number) => string,
  nextChangeAt: (tsMs: number) => number,
  tsMs: number
) => {
  const startMs = Date.now();
  const reading = read(tsMs);
  const changesAtMs = nextChangeAt(tsMs);
  if (changesAtMs === Number.POSITIVE_INFINITY) {
    // Far enough to cross a New Year from any start.
    for (const laterMs of [HOUR_MS, DAY_MS, 100 * DAY_MS, 400 * DAY_MS]) {
      vi.setSystemTime(startMs + laterMs);
      expect(read(tsMs)).toBe(reading);
    }
    return;
  }
  expect(changesAtMs).toBeGreaterThan(startMs);
  for (const beforeMs of [
    startMs + Math.floor((changesAtMs - startMs) / 2),
    Math.ceil(changesAtMs) - 1,
  ]) {
    vi.setSystemTime(beforeMs);
    expect(read(tsMs)).toBe(reading);
    expect(nextChangeAt(tsMs)).toBe(changesAtMs);
  }
  vi.setSystemTime(changesAtMs);
  expect(read(tsMs)).not.toBe(reading);
};

// Production timestamps carry sub-millisecond nanos, and `HumanizeTs` rounds
// them through seconds and back; whole-millisecond ones alone hide any
// boundary that lands a fraction early. The late-December start puts a New
// Year inside every window.
const BOUNDARY_STARTS = ["2026-03-02T12:00:00Z", "2026-12-31T15:30:00Z"];
const TIMESTAMP_FRACTIONS_MS = [0, 0.25, 0.999];

const boundaryCases = <T>(offsets: T[]) =>
  BOUNDARY_STARTS.flatMap((start) =>
    offsets.flatMap((offset) =>
      TIMESTAMP_FRACTIONS_MS.flatMap((fractionMs) => [
        { start, offset, fractionMs, viaSeconds: false },
        { start, offset, fractionMs, viaSeconds: true },
      ])
    )
  );

const startAt = (start: string) => {
  const startMs = new Date(start).getTime();
  vi.setSystemTime(startMs);
  return startMs;
};

const timestampOf = (
  instantMs: number,
  fractionMs: number,
  viaSeconds: boolean
) => {
  const tsMs = instantMs + fractionMs;
  return viaSeconds ? (tsMs / 1000) * 1000 : tsMs;
};

describe("reading boundaries", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test.each(boundaryCases(SAMPLED_AGES_MS))(
    "names the instant the work-queue reading changes (age $offset ms from $start, +$fractionMs ms, via seconds: $viaSeconds)",
    ({ start, offset, fractionMs, viaSeconds }) => {
      const startMs = startAt(start);
      expectBoundaryMatchesReading(
        formatQueueTime,
        nextQueueTimeChangeAt,
        timestampOf(startMs - offset, fractionMs, viaSeconds)
      );
    }
  );

  test.each(boundaryCases(SAMPLED_AGES_MS))(
    "names the instant the absolute date reading changes (age $offset ms from $start, +$fractionMs ms, via seconds: $viaSeconds)",
    ({ start, offset, fractionMs, viaSeconds }) => {
      const startMs = startAt(start);
      expectBoundaryMatchesReading(
        formatAbsoluteDate,
        nextAbsoluteDateChangeAt,
        timestampOf(startMs - offset, fractionMs, viaSeconds)
      );
    }
  );

  test.each(boundaryCases(SAMPLED_AGES_MS))(
    "names the instant the relative reading changes (age $offset ms from $start, +$fractionMs ms, via seconds: $viaSeconds)",
    ({ start, offset, fractionMs, viaSeconds }) => {
      const startMs = startAt(start);
      expectBoundaryMatchesReading(
        formatRelativeTime,
        nextRelativeTimeChangeAt,
        timestampOf(startMs - offset, fractionMs, viaSeconds)
      );
    }
  );
});

describe("deadline readings", () => {
  // Time left before a deadline, around each edge the readings have: passing,
  // every minute and the one-day line, and whole days beyond it.
  const SAMPLED_REMAINING_MS = [
    -DAY_MS,
    -1,
    0,
    1,
    30 * SECOND_MS,
    MINUTE_MS - 1,
    MINUTE_MS,
    MINUTE_MS + 1,
    90 * SECOND_MS,
    HOUR_MS - 1,
    HOUR_MS,
    3 * HOUR_MS + 20 * MINUTE_MS,
    DAY_MS - 1,
    DAY_MS,
    DAY_MS + 1,
    1.5 * DAY_MS,
    2 * DAY_MS,
    2 * DAY_MS + 1,
    45 * DAY_MS,
    400 * DAY_MS,
  ];

  const baseMs = new Date("2026-03-02T12:00:00Z").getTime();

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(baseMs);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const pairs = [
    [
      "countdown",
      (targetMs: number) => JSON.stringify(readCountdown(targetMs)),
      nextCountdownChangeAt,
    ],
    [
      "days-left",
      (targetMs: number) => JSON.stringify(readDaysLeft(targetMs)),
      nextDaysLeftChangeAt,
    ],
    ["passed", (targetMs: number) => String(hasPassed(targetMs)), nextPassedAt],
  ] as const;

  describe.each(pairs)("%s", (_name, read, nextChangeAt) => {
    test.each(boundaryCases(SAMPLED_REMAINING_MS))(
      "names the instant the reading changes ($offset ms left from $start, +$fractionMs ms, via seconds: $viaSeconds)",
      ({ start, offset, fractionMs, viaSeconds }) => {
        const startMs = startAt(start);
        expectBoundaryMatchesReading(
          read,
          nextChangeAt,
          timestampOf(startMs + offset, fractionMs, viaSeconds)
        );
      }
    );
  });

  test("counts down in hours and minutes within the last day", () => {
    expect(
      readCountdown(baseMs + 3 * HOUR_MS + 20 * MINUTE_MS + 59_999)
    ).toEqual({ kind: "within", hours: 3, minutes: 20 });
    expect(readCountdown(baseMs + DAY_MS)).toEqual({ kind: "beyondDay" });
    expect(readCountdown(baseMs)).toEqual({
      kind: "within",
      hours: 0,
      minutes: 0,
    });
    expect(readCountdown(baseMs - 1)).toEqual({ kind: "passed" });
  });

  test("rounds whole days up, and treats the deadline itself as passed", () => {
    expect(readDaysLeft(baseMs + DAY_MS + 1)).toEqual({
      kind: "days",
      days: 2,
    });
    expect(readDaysLeft(baseMs + DAY_MS)).toEqual({ kind: "days", days: 1 });
    expect(readDaysLeft(baseMs + DAY_MS - 1)).toEqual({ kind: "today" });
    expect(readDaysLeft(baseMs)).toEqual({ kind: "passed" });
  });
});
