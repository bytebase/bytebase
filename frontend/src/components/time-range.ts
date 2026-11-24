import dayjs from "dayjs";

/** A closed range of Unix milliseconds. */
export interface TimeRange {
  from: number;
  to: number;
}

export const QUICK_RANGE_IDS = [
  "today",
  "last-7-days",
  "last-30-days",
  "last-3-months",
  "last-6-months",
  "last-12-months",
] as const;
export type QuickRangeId = (typeof QUICK_RANGE_IDS)[number];

export const START_OF_DAY = "00:00";
export const END_OF_DAY = "23:59";

/**
 * Whole days through the end of today, counting today: "last 7 days" is today
 * and the 6 days before, and "last 3 months" on Nov 24 starts on Aug 25.
 */
export const quickRange = (
  id: QuickRangeId,
  now: Date = new Date()
): TimeRange => {
  const today = dayjs(now);
  const since = (from: dayjs.Dayjs): TimeRange => ({
    from: from.startOf("day").valueOf(),
    to: today.endOf("day").valueOf(),
  });
  const monthsBack = (months: number) =>
    since(today.subtract(months, "month").add(1, "day"));
  switch (id) {
    case "today":
      return since(today);
    case "last-7-days":
      return since(today.subtract(6, "day"));
    case "last-30-days":
      return since(today.subtract(29, "day"));
    case "last-3-months":
      return monthsBack(3);
    case "last-6-months":
      return monthsBack(6);
    case "last-12-months":
      return monthsBack(12);
  }
};

/** The quick range that `range` equals today, if any. */
export const matchQuickRange = (
  range: TimeRange,
  now: Date = new Date()
): QuickRangeId | undefined =>
  QUICK_RANGE_IDS.find((id) => {
    const quick = quickRange(id, now);
    return quick.from === range.from && quick.to === range.to;
  });

/** Reads "9", "930", "9:30" or "09.30" as "HH:mm". */
export const parseTimeOfDay = (text: string): string | undefined => {
  const value = text.trim();
  let hour: number;
  let minute: number;
  const separated = value.match(/^(\d{1,2})[:.](\d{1,2})$/);
  if (separated) {
    hour = Number(separated[1]);
    minute = Number(separated[2]);
  } else if (/^\d{3,4}$/.test(value)) {
    hour = Number(value.slice(0, -2));
    minute = Number(value.slice(-2));
  } else if (/^\d{1,2}$/.test(value)) {
    hour = Number(value);
    minute = 0;
  } else {
    return undefined;
  }
  if (hour > 23 || minute > 59) return undefined;
  return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
};

const atTime = (day: Date, time: string) => {
  const [hour, minute] = time.split(":").map(Number);
  return dayjs(day).hour(hour).minute(minute).second(0).millisecond(0);
};

export const rangeStart = (day: Date, time: string): number =>
  atTime(day, time).valueOf();

/** As the end of a range, 23:59 runs through the last millisecond of the day. */
export const rangeEnd = (day: Date, time: string): number =>
  time === END_OF_DAY
    ? dayjs(day).endOf("day").valueOf()
    : atTime(day, time).valueOf();

export const timeOfDay = (ts: number): string => dayjs(ts).format("HH:mm");

export const isWholeDays = (range: TimeRange): boolean =>
  timeOfDay(range.from) === START_OF_DAY && timeOfDay(range.to) === END_OF_DAY;

export const formatDay = (ts: number): string => dayjs(ts).format("YYYY-MM-DD");

export const parseRangeScope = (
  value: string | undefined
): TimeRange | undefined => {
  if (!value) return undefined;
  const parts = value.split(",");
  if (parts.length !== 2) return undefined;
  const from = parseInt(parts[0], 10);
  const to = parseInt(parts[1], 10);
  if (Number.isNaN(from) || Number.isNaN(to)) return undefined;
  return { from, to };
};

export const formatRangeScope = (range: TimeRange): string =>
  `${range.from},${range.to}`;
