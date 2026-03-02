import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("@/plugins/i18n", () => ({
  locale: { value: "en-US" },
}));

import {
  formatAbsoluteDate,
  formatAbsoluteDateTime,
  formatRelativeTime,
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
  test("includes month, day, year, and time", () => {
    const ts = new Date("2026-03-02T14:30:00Z").getTime();
    const result = formatAbsoluteDateTime(ts);
    expect(result).toContain("Mar");
    expect(result).toContain("2026");
    expect(result).toContain("2");
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

describe("humanizeTs", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("returns relative time for recent timestamps", async () => {
    const { humanizeTs } = await import("./util");
    const recentTs = Math.floor(Date.now() / 1000) - 3600; // 1 hour ago
    const result = humanizeTs(recentTs);
    expect(result).toMatch(/hour/);
  });

  test("returns absolute date for timestamps >30 days old", async () => {
    const { humanizeTs } = await import("./util");
    const oldTs =
      Math.floor(Date.now() / 1000) - RELATIVE_THRESHOLD_MS / 1000 - 86400; // 31 days ago
    const result = humanizeTs(oldTs);
    expect(result).not.toMatch(/ago/);
    expect(result).toContain("Jan");
  });
});
