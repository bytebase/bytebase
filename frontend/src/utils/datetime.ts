import i18n from "@/lib/i18n";

export const RELATIVE_THRESHOLD_MS = 30 * 24 * 60 * 60 * 1000; // 30 days
export const DEFAULT_NOW_THRESHOLD_MS = 10_000;

type RelativeTimeFormatOptions = {
  nowThresholdMs?: number;
};

export function getActiveLocale(): string {
  return i18n.language;
}

export function formatRelativeTime(
  timestampMs: number,
  options: RelativeTimeFormatOptions = {}
): string {
  const { nowThresholdMs = DEFAULT_NOW_THRESHOLD_MS } = options;
  const diffMs = Date.now() - timestampMs;
  const absDiff = Math.abs(diffMs);
  const sign = diffMs >= 0 ? -1 : 1;

  const rtf = new Intl.RelativeTimeFormat(getActiveLocale(), {
    numeric: "auto",
  });

  if (absDiff < nowThresholdMs) {
    return rtf.format(0, "second");
  }
  if (absDiff < 60_000) {
    return rtf.format(sign * Math.round(absDiff / 1000), "second");
  }
  if (absDiff < 3_600_000) {
    return rtf.format(sign * Math.round(absDiff / 60_000), "minute");
  }
  if (absDiff < 86_400_000) {
    return rtf.format(sign * Math.round(absDiff / 3_600_000), "hour");
  }
  return rtf.format(sign * Math.round(absDiff / 86_400_000), "day");
}

export function formatAbsoluteDateTime(timestampMs: number): string {
  return new Intl.DateTimeFormat(getActiveLocale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
  }).format(new Date(timestampMs));
}

export function formatAbsoluteDate(timestampMs: number): string {
  const date = new Date(timestampMs);
  const now = new Date();

  if (date.getFullYear() === now.getFullYear()) {
    return new Intl.DateTimeFormat(getActiveLocale(), {
      month: "short",
      day: "numeric",
    }).format(date);
  }

  return new Intl.DateTimeFormat(getActiveLocale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(date);
}

export function formatQueueTime(timestampMs: number): string {
  if (Math.abs(Date.now() - timestampMs) >= RELATIVE_THRESHOLD_MS) {
    return formatAbsoluteDate(timestampMs);
  }
  return formatRelativeTime(timestampMs);
}

export function formatCompactDateTime(timestampMs: number): string {
  return new Intl.DateTimeFormat(getActiveLocale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(timestampMs));
}

export function formatOperationalDateTime(timestampMs: number): string {
  return new Intl.DateTimeFormat(getActiveLocale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(new Date(timestampMs));
}

/**
 * The instant at which `formatQueueTime` would next render this timestamp
 * differently, or `Infinity` once it has settled on an absolute date.
 *
 * Buckets are measured from the timestamp, not from the wall clock: a row
 * created at 10:00:30 turns over to "1 minute ago" at 10:01:30. A display that
 * woke on a fixed cadence instead would lag by up to a whole bucket.
 */
export function nextRelativeChangeAt(timestampMs: number): number {
  const nowMs = Date.now();
  const ageMs = Math.abs(nowMs - timestampMs);
  if (ageMs >= RELATIVE_THRESHOLD_MS) {
    return Number.POSITIVE_INFINITY;
  }
  const bucketMs =
    ageMs < 60_000
      ? 1_000
      : ageMs < 3_600_000
        ? 60_000
        : ageMs < 86_400_000
          ? 3_600_000
          : 86_400_000;
  // A past timestamp grows into the next bucket; a future one shrinks back
  // into the previous one.
  const intoBucketMs = ageMs % bucketMs;
  const untilBucketEdgeMs =
    nowMs >= timestampMs ? bucketMs - intoBucketMs : intoBucketMs || bucketMs;
  return nowMs + Math.min(untilBucketEdgeMs, RELATIVE_THRESHOLD_MS - ageMs);
}
