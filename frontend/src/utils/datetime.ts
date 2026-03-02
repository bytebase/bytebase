import { locale } from "@/plugins/i18n";

export const RELATIVE_THRESHOLD_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

export function getActiveLocale(): string {
  return locale.value;
}

export function formatRelativeTime(timestampMs: number): string {
  const diffMs = Date.now() - timestampMs;
  const absDiff = Math.abs(diffMs);
  const sign = diffMs >= 0 ? -1 : 1;

  const rtf = new Intl.RelativeTimeFormat(getActiveLocale(), {
    numeric: "auto",
  });

  if (absDiff < 10_000) {
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
