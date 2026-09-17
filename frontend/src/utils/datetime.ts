import i18n from "@/lib/i18n";

const SECOND_MS = 1_000;
const MINUTE_MS = 60 * SECOND_MS;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

export const RELATIVE_THRESHOLD_MS = 30 * DAY_MS;
export const DEFAULT_NOW_THRESHOLD_MS = 10_000;

type RelativeTimeFormatOptions = {
  nowThresholdMs?: number;
};

export function getActiveLocale(): string {
  return i18n.language;
}

type RelativeUnit = {
  fromMs: number;
  unitMs: number;
  unit: Intl.RelativeTimeFormatUnit;
};

// The units a relative reading counts in, each from the age it takes over at.
// Below the first, the reading is "now". Both `formatRelativeTime` and the
// boundaries that schedule it read this table, so the two cannot drift apart.
const relativeUnits = (nowThresholdMs: number): RelativeUnit[] => [
  { fromMs: nowThresholdMs, unitMs: SECOND_MS, unit: "second" },
  { fromMs: MINUTE_MS, unitMs: MINUTE_MS, unit: "minute" },
  { fromMs: HOUR_MS, unitMs: HOUR_MS, unit: "hour" },
  { fromMs: DAY_MS, unitMs: DAY_MS, unit: "day" },
];

const unitIndexForAge = (units: RelativeUnit[], ageMs: number): number =>
  units.findLastIndex((unit) => ageMs >= unit.fromMs);

// Building an Intl formatter costs tens of times more than formatting with one,
// and these run in every table cell, so each is built once per locale. The time
// zone is fixed when a formatter is built: a zone change mid-session shows on
// reload, and every string that names its zone names the one it was built in.
const formatterCache = new Map<string, unknown>();

function cachedFormatter<T>(name: string, build: (locale: string) => T): T {
  const locale = getActiveLocale();
  const key = `${name}|${locale}`;
  let formatter = formatterCache.get(key) as T | undefined;
  if (formatter === undefined) {
    formatter = build(locale);
    formatterCache.set(key, formatter);
  }
  return formatter;
}

const dateTimeFormatter = (
  name: string,
  options: Intl.DateTimeFormatOptions
): Intl.DateTimeFormat =>
  cachedFormatter(name, (locale) => new Intl.DateTimeFormat(locale, options));

export function formatRelativeTime(
  timestampMs: number,
  options: RelativeTimeFormatOptions = {}
): string {
  const { nowThresholdMs = DEFAULT_NOW_THRESHOLD_MS } = options;
  const diffMs = Date.now() - timestampMs;
  const ageMs = Math.abs(diffMs);
  const rtf = cachedFormatter(
    "relative",
    (locale) => new Intl.RelativeTimeFormat(locale, { numeric: "auto" })
  );

  const units = relativeUnits(nowThresholdMs);
  const index = unitIndexForAge(units, ageMs);
  if (index < 0) {
    return rtf.format(0, "second");
  }
  const { unitMs, unit } = units[index];
  const sign = diffMs >= 0 ? -1 : 1;
  return rtf.format(sign * Math.round(ageMs / unitMs), unit);
}

export function formatAbsoluteDateTime(timestampMs: number): string {
  return dateTimeFormatter("datetime", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
  }).format(timestampMs);
}

export function formatAbsoluteDate(timestampMs: number): string {
  if (new Date(timestampMs).getFullYear() === new Date().getFullYear()) {
    return dateTimeFormatter("date", {
      month: "short",
      day: "numeric",
    }).format(timestampMs);
  }
  return dateTimeFormatter("dateWithYear", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(timestampMs);
}

export function formatQueueTime(timestampMs: number): string {
  if (Math.abs(Date.now() - timestampMs) >= RELATIVE_THRESHOLD_MS) {
    return formatAbsoluteDate(timestampMs);
  }
  return formatRelativeTime(timestampMs);
}

export function formatCompactDateTime(timestampMs: number): string {
  return dateTimeFormatter("compact", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(timestampMs);
}

export function formatOperationalDateTime(timestampMs: number): string {
  return dateTimeFormatter("operational", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(timestampMs);
}

/**
 * The ages bracketing `ageMs` at which a relative reading changes: the latest
 * at or below it and the earliest above it. A reading changes where its unit
 * switches and, because the count is rounded, at every half unit in between.
 * At `capMs` it stops counting altogether.
 */
function relativeChangePoints(
  ageMs: number,
  nowThresholdMs: number,
  capMs: number
): { atOrBelowMs?: number; aboveMs: number } {
  if (ageMs >= capMs) {
    return { atOrBelowMs: capMs, aboveMs: Number.POSITIVE_INFINITY };
  }
  const units = relativeUnits(nowThresholdMs);
  const index = unitIndexForAge(units, ageMs);
  if (index < 0) {
    return { aboveMs: Math.min(units[0].fromMs, capMs) };
  }
  const { fromMs, unitMs } = units[index];
  const toMs = Math.min(
    units[index + 1]?.fromMs ?? Number.POSITIVE_INFINITY,
    capMs
  );
  return {
    atOrBelowMs: Math.max(
      (Math.floor(ageMs / unitMs - 0.5) + 0.5) * unitMs,
      fromMs
    ),
    aboveMs: Math.min((Math.floor(ageMs / unitMs + 0.5) + 0.5) * unitMs, toMs),
  };
}

function nextRelativeReadingChangeAt(
  timestampMs: number,
  nowThresholdMs: number,
  capMs: number
): number {
  const diffMs = Date.now() - timestampMs;
  const { atOrBelowMs, aboveMs } = relativeChangePoints(
    Math.abs(diffMs),
    nowThresholdMs,
    capMs
  );
  if (diffMs >= 0) {
    return Math.ceil(timestampMs + aboveMs);
  }
  // A future timestamp counts down, so its reading changes the moment its age
  // drops below the change point at or under it. Inside the "now" window there
  // is none: "now" holds across the timestamp until the past side leaves it.
  if (atOrBelowMs === undefined) {
    return Math.ceil(timestampMs + nowThresholdMs);
  }
  return Math.floor(timestampMs - atOrBelowMs) + 1;
}

/**
 * The first instant `formatQueueTime` renders this timestamp differently, or
 * `Infinity` once it has settled on a date for good.
 */
export function nextQueueTimeChangeAt(timestampMs: number): number {
  const changesAtMs = nextRelativeReadingChangeAt(
    timestampMs,
    DEFAULT_NOW_THRESHOLD_MS,
    RELATIVE_THRESHOLD_MS
  );
  if (Math.abs(Date.now() - timestampMs) < RELATIVE_THRESHOLD_MS) {
    return changesAtMs;
  }
  // A date shows its year unless it falls in the current one, so the reading
  // also changes when the current year turns into or out of the timestamp's.
  const timestampYear = new Date(timestampMs).getFullYear();
  const yearTurnsMs = [timestampYear, timestampYear + 1]
    .map((year) => new Date(year, 0, 1).getTime())
    .find((yearStartMs) => yearStartMs > Date.now());
  return Math.min(changesAtMs, yearTurnsMs ?? Number.POSITIVE_INFINITY);
}

/** The first instant `formatRelativeTime` renders this timestamp differently. */
export function nextRelativeTimeChangeAt(
  timestampMs: number,
  options: RelativeTimeFormatOptions = {}
): number {
  return nextRelativeReadingChangeAt(
    timestampMs,
    options.nowThresholdMs ?? DEFAULT_NOW_THRESHOLD_MS,
    Number.POSITIVE_INFINITY
  );
}

// Readings of a deadline. Each is paired with the first instant it changes, so
// a display showing one stays current on the shared clock.

/** Whether a deadline is behind us; the deadline instant itself is not. */
export function hasPassed(targetMs: number): boolean {
  return targetMs < Date.now();
}

export function nextPassedAt(targetMs: number): number {
  return hasPassed(targetMs)
    ? Number.POSITIVE_INFINITY
    : Math.floor(targetMs) + 1;
}

/**
 * Time left before a deadline as a countdown: hours and whole minutes within
 * the last day, and no count beyond it.
 */
export type Countdown =
  | { kind: "passed" }
  | { kind: "beyondDay" }
  | { kind: "within"; hours: number; minutes: number };

export function readCountdown(targetMs: number): Countdown {
  const remainingMs = targetMs - Date.now();
  if (remainingMs < 0) {
    return { kind: "passed" };
  }
  if (remainingMs >= DAY_MS) {
    return { kind: "beyondDay" };
  }
  return {
    kind: "within",
    hours: Math.floor(remainingMs / HOUR_MS),
    minutes: Math.floor((remainingMs % HOUR_MS) / MINUTE_MS),
  };
}

export function nextCountdownChangeAt(targetMs: number): number {
  const remainingMs = targetMs - Date.now();
  if (remainingMs < 0) {
    return Number.POSITIVE_INFINITY;
  }
  // The count is whole minutes left, so it drops the moment the time left
  // falls below the current count. With no minute left, that is the deadline.
  const thresholdMs =
    remainingMs >= DAY_MS
      ? DAY_MS
      : Math.floor(remainingMs / MINUTE_MS) * MINUTE_MS;
  return Math.floor(targetMs - thresholdMs) + 1;
}

/**
 * Time left before a deadline in whole days, rounded up; "today" within the
 * last day. Unlike `hasPassed`, the deadline instant itself counts as passed.
 */
export type DaysLeft =
  | { kind: "passed" }
  | { kind: "today" }
  | { kind: "days"; days: number };

export function readDaysLeft(targetMs: number): DaysLeft {
  const remainingMs = targetMs - Date.now();
  if (remainingMs <= 0) {
    return { kind: "passed" };
  }
  if (remainingMs < DAY_MS) {
    return { kind: "today" };
  }
  return { kind: "days", days: Math.ceil(remainingMs / DAY_MS) };
}

export function nextDaysLeftChangeAt(targetMs: number): number {
  const remainingMs = targetMs - Date.now();
  if (remainingMs <= 0) {
    return Number.POSITIVE_INFINITY;
  }
  if (remainingMs < DAY_MS) {
    return Math.ceil(targetMs);
  }
  // Rounded up, the count drops once the time left reaches the count below
  // it — except the last whole day, which turns into "today" only once less
  // than a day remains.
  const days = Math.ceil(remainingMs / DAY_MS);
  return days === 1
    ? Math.floor(targetMs - DAY_MS) + 1
    : Math.ceil(targetMs - (days - 1) * DAY_MS);
}
