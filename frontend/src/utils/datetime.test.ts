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
  absoluteTimeReading,
  compactTimeReading,
  countdownReading,
  daysLeftReading,
  displayableInstantMs,
  formatAbsoluteDateTime,
  operationalTimeReading,
  passedReading,
  queueTimeReading,
  relativeTimeReading,
  type TimeReading,
} from "./datetime";

const SECOND_MS = 1_000;
const MINUTE_MS = 60_000;
const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;
// The doc's threshold, spelled out rather than imported: a test that borrows
// the value it checks cannot catch the value changing.
const RELATIVE_THRESHOLD_MS = 30 * DAY_MS;

describe("the relative reading", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("returns 'now' for timestamps less than 10 seconds ago", () => {
    const result = relativeTimeReading.read(Date.now() - 5000);
    expect(result).toBe("now");
  });

  test("returns 'X seconds ago' for 10-59 seconds", () => {
    const result = relativeTimeReading.read(Date.now() - 30_000);
    expect(result).toContain("seconds ago");
  });

  test("returns 'X minutes ago' for 1-59 minutes", () => {
    const result = relativeTimeReading.read(Date.now() - 5 * 60_000);
    expect(result).toMatch(/minutes? ago/);
  });

  test("returns 'X hours ago' for 1-23 hours", () => {
    const result = relativeTimeReading.read(Date.now() - 3 * 3_600_000);
    expect(result).toMatch(/hours? ago/);
  });

  test("returns 'yesterday' for ~24 hours ago", () => {
    const result = relativeTimeReading.read(Date.now() - 24 * 3_600_000);
    expect(result).toBe("yesterday");
  });

  test("returns 'X days ago' for 2-30 days", () => {
    const result = relativeTimeReading.read(Date.now() - 10 * 86_400_000);
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

describe("the work-queue reading", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("reads as relative age inside the 30-day window", () => {
    expect(queueTimeReading.read(Date.now() - 6 * 86_400_000)).toBe(
      "6 days ago"
    );
  });

  test("still reads as relative age one day short of the threshold", () => {
    expect(queueTimeReading.read(Date.now() - 29 * 86_400_000)).toBe(
      "29 days ago"
    );
  });

  test("switches to an absolute date at the threshold", () => {
    expect(queueTimeReading.read(Date.now() - RELATIVE_THRESHOLD_MS)).toBe(
      "Jan 31"
    );
  });

  test("keeps the year on a date from another year", () => {
    expect(queueTimeReading.read(Date.now() - 90 * 86_400_000)).toBe(
      "Dec 2, 2025"
    );
  });

  test("measures a future timestamp by the same threshold", () => {
    expect(queueTimeReading.read(Date.now() + 2 * 86_400_000)).toBe(
      "in 2 days"
    );
    expect(queueTimeReading.read(Date.now() + 40 * 86_400_000)).toBe("Apr 11");
  });

  test("renders both branches in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(queueTimeReading.read(Date.now() - 6 * 86_400_000)).toBe("6天前");
      expect(queueTimeReading.read(Date.now() - RELATIVE_THRESHOLD_MS)).toBe(
        "1月31日"
      );
    });
  });
});

describe("the compact reading", () => {
  const ts = new Date("2026-08-26T06:03:22Z").getTime();

  test("drops the seconds and the timezone a history row does not need", () => {
    expect(compactTimeReading.read(ts)).toBe("Aug 26, 2026, 2:03 PM");
  });

  test("renders in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(compactTimeReading.read(ts)).toBe("2026年8月26日 14:03");
    });
  });
});

describe("the operational reading", () => {
  // The value a reader is about to act on: 9am Shanghai, with a sub-minute
  // tail of the kind a `now() + N days` expiration preset writes.
  const ts = new Date("2026-09-15T01:00:22Z").getTime();

  test("names the timezone in the visible string", () => {
    expect(operationalTimeReading.read(ts)).toBe("Sep 15, 2026, 9:00 AM GMT+8");
  });

  test("floors to the minute rather than showing the seconds", () => {
    expect(operationalTimeReading.read(ts)).toBe(
      operationalTimeReading.read(new Date("2026-09-15T01:00:00Z").getTime())
    );
  });

  test("renders in the active locale", () => {
    withLocale("zh-CN", () => {
      expect(operationalTimeReading.read(ts)).toBe("2026年9月15日 GMT+8 09:00");
    });
  });
});

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
// holds throughout the span up to the named instant, the boundary names that
// same instant from anywhere inside the span, and the reading changes at it.
// Each check follows the chain for several links, starting every link where
// the last one ended, so a reading that leaves a value and later returns to it
// cannot hide a skipped span. A boundary checked on its own cannot see that it
// was derived from the wrong reading.
const CHAIN_LINKS = 3;
const SPAN_PROBES = [0.25, 0.5, 0.75];
// A reading with no boundary is sampled out past a New Year from any start.
const FINAL_PROBES_MS = [
  HOUR_MS,
  DAY_MS,
  7 * DAY_MS,
  15 * DAY_MS,
  30 * DAY_MS,
  60 * DAY_MS,
  200 * DAY_MS,
  400 * DAY_MS,
];

const expectBoundaryMatchesReading = <Value>(
  reading: TimeReading<number, Value>,
  tsMs: number
) => {
  const read = (input: number) => JSON.stringify(reading.read(input));
  const { nextChangeAt } = reading;
  for (let link = 0; link < CHAIN_LINKS; link++) {
    const startMs = Date.now();
    const value = read(tsMs);
    const changesAtMs = nextChangeAt(tsMs);
    if (changesAtMs === Number.POSITIVE_INFINITY) {
      for (const laterMs of FINAL_PROBES_MS) {
        vi.setSystemTime(startMs + laterMs);
        expect(read(tsMs)).toBe(value);
      }
      return;
    }
    expect(changesAtMs).toBeGreaterThan(startMs);
    const spanMs = changesAtMs - startMs;
    for (const probeMs of [
      ...SPAN_PROBES.map((at) => startMs + Math.floor(spanMs * at)),
      Math.ceil(changesAtMs) - 1,
    ]) {
      vi.setSystemTime(probeMs);
      expect(read(tsMs)).toBe(value);
      expect(nextChangeAt(tsMs)).toBe(changesAtMs);
    }
    vi.setSystemTime(changesAtMs);
    expect(read(tsMs)).not.toBe(value);
  }
};

// Production timestamps carry sub-millisecond nanos, which hide a boundary
// that lands a fraction early; a start off the whole minute exposes a boundary
// computed from a rounded clock; the other two starts sit just before and
// exactly on the pinned zone's New Year.
const BOUNDARY_STARTS = [
  "2026-03-02T12:00:17.371Z",
  "2026-12-31T15:30:00Z",
  "2026-12-31T16:00:00Z",
];
const TIMESTAMP_FRACTIONS_MS = [0, 0.25, 0.999];

const boundaryCases = <T>(offsets: T[]) =>
  BOUNDARY_STARTS.flatMap((start) =>
    offsets.flatMap((offset) =>
      TIMESTAMP_FRACTIONS_MS.map((fractionMs) => ({
        start,
        offset,
        fractionMs,
      }))
    )
  );

const startAt = (start: string) => {
  const startMs = new Date(start).getTime();
  vi.setSystemTime(startMs);
  return startMs;
};

describe("reading boundaries", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe.each([
    ["work-queue", queueTimeReading],
    ["relative", relativeTimeReading],
  ])("%s", (_name, reading) => {
    test.each(boundaryCases(SAMPLED_AGES_MS))(
      "names the instant the reading changes (age $offset ms from $start, +$fractionMs ms)",
      ({ start, offset, fractionMs }) => {
        const startMs = startAt(start);
        expectBoundaryMatchesReading(reading, startMs - offset + fractionMs);
      }
    );
  });

  // A constant rendering names no instant at all, which is what keeps every
  // absolute timestamp in every table off the shared clock. Sampled rather
  // than swept: the ages only matter to a reading that changes with them, and
  // all of these take the same branch.
  describe.each([
    ["absolute", absoluteTimeReading],
    ["compact", compactTimeReading],
    ["operational", operationalTimeReading],
  ])("%s", (_name, reading) => {
    test.each([
      ["moments ago", SECOND_MS],
      ["within the day", 5 * HOUR_MS],
      ["past the relative threshold", 40 * DAY_MS],
    ])("never changes, %s", (_label, ageMs) => {
      const startMs = startAt(BOUNDARY_STARTS[0]);
      expectBoundaryMatchesReading(reading, startMs - ageMs);
    });
  });
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

  const readings = [
    ["countdown", countdownReading],
    ["days-left", daysLeftReading],
    ["passed", passedReading],
  ] as const;

  describe.each(readings)("%s", (_name, reading) => {
    test.each(boundaryCases(SAMPLED_REMAINING_MS))(
      "names the instant the reading changes ($offset ms left from $start, +$fractionMs ms)",
      ({ start, offset, fractionMs }) => {
        const startMs = startAt(start);
        expectBoundaryMatchesReading(
          reading as TimeReading<number, unknown>,
          startMs + offset + fractionMs
        );
      }
    );
  });

  test("counts down in hours and minutes within the last day", () => {
    expect(
      countdownReading.read(baseMs + 3 * HOUR_MS + 20 * MINUTE_MS + 59_999)
    ).toEqual({ kind: "within", hours: 3, minutes: 20 });
    // Past the half hour, where an hour rounded rather than floored reads as
    // the hour after the one that is left.
    expect(countdownReading.read(baseMs + HOUR_MS + 30 * MINUTE_MS)).toEqual({
      kind: "within",
      hours: 1,
      minutes: 30,
    });
    expect(countdownReading.read(baseMs + DAY_MS)).toEqual({
      kind: "beyondDay",
    });
    expect(countdownReading.read(baseMs)).toEqual({
      kind: "within",
      hours: 0,
      minutes: 0,
    });
    expect(countdownReading.read(baseMs - 1)).toEqual({ kind: "passed" });
  });

  test("rounds whole days up, and treats the deadline itself as passed", () => {
    expect(daysLeftReading.read(baseMs + DAY_MS + 1)).toEqual({
      kind: "days",
      days: 2,
    });
    expect(daysLeftReading.read(baseMs + DAY_MS)).toEqual({
      kind: "days",
      days: 1,
    });
    expect(daysLeftReading.read(baseMs + DAY_MS - 1)).toEqual({
      kind: "today",
    });
    expect(daysLeftReading.read(baseMs)).toEqual({ kind: "passed" });
  });
});

describe("displayableInstantMs", () => {
  // Every value a display can be handed, at and just past the edge of the
  // range a time value can occupy.
  const candidates = [
    0,
    -1,
    1_772_452_800_123,
    1_772_452_800_123.5,
    8.64e15,
    -8.64e15,
    8.64e15 + 1,
    -8.64e15 - 1,
    1e18,
    -1e18,
    Number.NaN,
    Number.POSITIVE_INFINITY,
    Number.NEGATIVE_INFINITY,
  ];

  test.each(candidates)(
    "accepts %p exactly when it is a time at all",
    (value) => {
      const isATime = !Number.isNaN(new Date(value).getTime());
      expect(displayableInstantMs(value)).toBe(isATime ? value : undefined);
    }
  );

  test.each(candidates)(
    "never yields %p to a formatter that throws",
    (value) => {
      if (displayableInstantMs(value) === undefined) {
        return;
      }
      // A value this admits reaches Intl, which throws on one outside the range.
      expect(() => formatAbsoluteDateTime(value)).not.toThrow();
      expect(() => compactTimeReading.read(value)).not.toThrow();
      expect(() => operationalTimeReading.read(value)).not.toThrow();
      expect(() => queueTimeReading.read(value)).not.toThrow();
    }
  );
});
