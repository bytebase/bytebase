// Layout for the forms HumanizeTs renders, kept out of the component so a
// test that stubs the component still has the real widths to lay out with.

/**
 * The width a table column opens at to hold a form on one line: the widest
 * rendering across every time zone and shipped locale at a table's 14px, plus
 * a cell's 32px of padding. For the forms that show a zone that is en-US in
 * Chatham, GMT+12:45; for compact, en-US at a two-digit hour in any zone.
 * Measured with the macOS system font; Segoe UI is not.
 */
export const TIMESTAMP_COLUMN_WIDTH = {
  compact: 192,
  operational: 270,
  datetime: 292,
} as const;

/**
 * How far any timestamp column may narrow, measured the same way: the date
 * whole, the space where the time meets it, and a whole ellipsis saying the
 * time is there. ja-JP and zh-CN need the most. `HumanizeTs`'s `truncate`
 * is what keeps the date whole.
 */
export const TIMESTAMP_COLUMN_MIN_WIDTH = 154;
