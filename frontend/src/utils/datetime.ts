import i18n from "@/lib/i18n";

const SECOND_MS = 1_000;
const MINUTE_MS = 60 * SECOND_MS;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;

export const RELATIVE_THRESHOLD_MS = 30 * DAY_MS;
const NOW_THRESHOLD_MS = 10 * SECOND_MS;

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
// boundary that schedules it read this table, so the two cannot drift apart.
const RELATIVE_UNITS: readonly RelativeUnit[] = [
  { fromMs: NOW_THRESHOLD_MS, unitMs: SECOND_MS, unit: "second" },
  { fromMs: MINUTE_MS, unitMs: MINUTE_MS, unit: "minute" },
  { fromMs: HOUR_MS, unitMs: HOUR_MS, unit: "hour" },
  { fromMs: DAY_MS, unitMs: DAY_MS, unit: "day" },
];

const unitIndexForAge = (ageMs: number): number =>
  RELATIVE_UNITS.findLastIndex((unit) => ageMs >= unit.fromMs);

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

/**
 * A value that changes with time, paired with the first instant it changes
 * (`Infinity` if never). Displays read one only through `useTimeReading`,
 * which keeps the two together and schedules the display on the shared clock.
 */
export type TimeReading<Input, Value> = {
  read: (input: Input) => Value;
  nextChangeAt: (input: Input) => number;
};

function formatRelativeTime(timestampMs: number): string {
  const diffMs = Date.now() - timestampMs;
  const ageMs = Math.abs(diffMs);
  const rtf = cachedFormatter(
    "relative",
    (locale) => new Intl.RelativeTimeFormat(locale, { numeric: "auto" })
  );

  const index = unitIndexForAge(ageMs);
  if (index < 0) {
    return rtf.format(0, "second");
  }
  const { unitMs, unit } = RELATIVE_UNITS[index];
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

function formatAbsoluteDate(timestampMs: number): string {
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

function formatQueueTime(timestampMs: number): string {
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

function relativeTimeChangeAt(timestampMs: number, nowMs: number): number {
  const diffMs = nowMs - timestampMs;
  const ageMs = Math.abs(diffMs);
  const index = unitIndexForAge(ageMs);
  // Inside the "now" window the reading holds across the timestamp itself,
  // until the past side leaves the window.
  if (index < 0) {
    return Math.ceil(timestampMs + NOW_THRESHOLD_MS);
  }
  // The reading prints a rounded count, so it changes at every half unit as
  // well as where the unit switches.
  const { fromMs, unitMs } = RELATIVE_UNITS[index];
  const count = Math.round(ageMs / unitMs);
  if (diffMs >= 0) {
    const toMs = RELATIVE_UNITS[index + 1]?.fromMs ?? Number.POSITIVE_INFINITY;
    return Math.ceil(timestampMs + Math.min((count + 0.5) * unitMs, toMs));
  }
  // A future timestamp counts down: its reading changes the moment its age
  // drops below the change point at or under it.
  return Math.floor(timestampMs - Math.max((count - 0.5) * unitMs, fromMs)) + 1;
}

function absoluteDateChangeAt(timestampMs: number, nowMs: number): number {
  // A date shows its year unless it falls in the current one, so it changes
  // when the current year turns into or out of the timestamp's.
  const year = new Date(timestampMs).getFullYear();
  return (
    [year, year + 1]
      .map((y) => new Date(y, 0, 1).getTime())
      .find((yearStartMs) => yearStartMs > nowMs) ?? Number.POSITIVE_INFINITY
  );
}

function nextRelativeTimeChangeAt(timestampMs: number): number {
  return relativeTimeChangeAt(timestampMs, Date.now());
}

/** A timestamp's age, as "5 minutes ago" or "in 2 days". */
export const relativeTimeReading: TimeReading<number, string> = {
  read: formatRelativeTime,
  nextChangeAt: nextRelativeTimeChangeAt,
};

function nextQueueTimeChangeAt(timestampMs: number): number {
  const nowMs = Date.now();
  const diffMs = nowMs - timestampMs;
  if (Math.abs(diffMs) < RELATIVE_THRESHOLD_MS) {
    // A future timestamp's relative boundary always comes before the switch.
    return Math.min(
      relativeTimeChangeAt(timestampMs, nowMs),
      Math.ceil(timestampMs + RELATIVE_THRESHOLD_MS)
    );
  }
  // Past the switch it reads as a date, and a future one counts back into
  // the relative window.
  const reentersAtMs =
    diffMs < 0
      ? Math.floor(timestampMs - RELATIVE_THRESHOLD_MS) + 1
      : Number.POSITIVE_INFINITY;
  return Math.min(reentersAtMs, absoluteDateChangeAt(timestampMs, nowMs));
}

/**
 * A work-queue timestamp: its age within 30 days, a date beyond. Settles for
 * good once it reads as a date from another year.
 */
export const queueTimeReading: TimeReading<number, string> = {
  read: formatQueueTime,
  nextChangeAt: nextQueueTimeChangeAt,
};

// Readings of a deadline.

function hasPassed(targetMs: number): boolean {
  return targetMs < Date.now();
}

function nextPassedAt(targetMs: number): number {
  return hasPassed(targetMs)
    ? Number.POSITIVE_INFINITY
    : Math.floor(targetMs) + 1;
}

/** Whether a deadline is behind us; the deadline instant itself is not. */
export const passedReading: TimeReading<number, boolean> = {
  read: hasPassed,
  nextChangeAt: nextPassedAt,
};

/**
 * Time left before a deadline as a countdown: hours and whole minutes within
 * the last day, and no count beyond it.
 */
export type Countdown =
  | { kind: "passed" }
  | { kind: "beyondDay" }
  | { kind: "within"; hours: number; minutes: number };

function readCountdown(targetMs: number): Countdown {
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

function nextCountdownChangeAt(targetMs: number): number {
  const remainingMs = targetMs - Date.now();
  if (remainingMs < 0) {
    return Number.POSITIVE_INFINITY;
  }
  // The count is whole minutes left, so it drops the moment the time left
  // falls below the current count. With no minute left, that is the deadline.
  const thresholdMs = Math.min(
    Math.floor(remainingMs / MINUTE_MS) * MINUTE_MS,
    DAY_MS
  );
  return Math.floor(targetMs - thresholdMs) + 1;
}

export const countdownReading: TimeReading<number, Countdown> = {
  read: readCountdown,
  nextChangeAt: nextCountdownChangeAt,
};

/**
 * Time left before a deadline in whole days, rounded up; "today" within the
 * last day. Unlike `hasPassed`, the deadline instant itself counts as passed.
 */
export type DaysLeft =
  | { kind: "passed" }
  | { kind: "today" }
  | { kind: "days"; days: number };

function readDaysLeft(targetMs: number): DaysLeft {
  const remainingMs = targetMs - Date.now();
  if (remainingMs <= 0) {
    return { kind: "passed" };
  }
  if (remainingMs < DAY_MS) {
    return { kind: "today" };
  }
  return { kind: "days", days: Math.ceil(remainingMs / DAY_MS) };
}

function nextDaysLeftChangeAt(targetMs: number): number {
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

export const daysLeftReading: TimeReading<number, DaysLeft> = {
  read: readDaysLeft,
  nextChangeAt: nextDaysLeftChangeAt,
};
