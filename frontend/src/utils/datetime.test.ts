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
  nextRelativeChangeAt,
  RELATIVE_THRESHOLD_MS,
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

  test("supports overriding the 'now' threshold", () => {
    const result = formatRelativeTime(Date.now() - 5000, {
      nowThresholdMs: 3000,
    });
    expect(result).toContain("seconds ago");
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

describe("nextRelativeChangeAt", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("tracks the second while the label still counts seconds", () => {
    expect(nextRelativeChangeAt(Date.now() - 30_000)).toBe(Date.now() + 1_000);
  });

  test("waits for the turn of the minute, not a fixed minute from now", () => {
    expect(nextRelativeChangeAt(Date.now() - 90_000)).toBe(Date.now() + 30_000);
  });

  test("counts a future timestamp down to the same boundary", () => {
    expect(nextRelativeChangeAt(Date.now() + 90_000)).toBe(Date.now() + 30_000);
  });

  test("wakes at the 30-day switch rather than at the next day", () => {
    const ts = Date.now() - (RELATIVE_THRESHOLD_MS - 3_600_000);
    expect(nextRelativeChangeAt(ts)).toBe(Date.now() + 3_600_000);
  });

  test.each([-90_000, 90_000])(
    "names the same instant on every render inside a bucket (offset %i)",
    (offsetMs) => {
      const ts = Date.now() + offsetMs;
      const first = nextRelativeChangeAt(ts);

      // A value that drifted with the clock would re-key the subscription on
      // every render, so the shared clock would thrash through a list.
      vi.advanceTimersByTime(100);
      expect(nextRelativeChangeAt(ts)).toBe(first);
    }
  );

  test("never wakes for a label already showing an absolute date", () => {
    const ts = Date.now() - RELATIVE_THRESHOLD_MS - 86_400_000;
    expect(nextRelativeChangeAt(ts)).toBe(Number.POSITIVE_INFINITY);
  });
});
