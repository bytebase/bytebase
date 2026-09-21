// Layout for the forms HumanizeTs renders, kept out of the component so a
// test that stubs the component still has the real widths to lay out with.

/**
 * The width a table column opens at to hold a form on one line: the widest
 * rendering across every time zone and shipped locale at a table's 14px, plus
 * a cell's 32px of padding. en-US in Chatham, GMT+12:45, is the widest in all
 * three forms. Measured with the macOS system font; Segoe UI is not.
 */
export const TIMESTAMP_COLUMN_WIDTH = {
  compact: 192,
  operational: 270,
  datetime: 292,
} as const;

/**
 * How far a reader may narrow any timestamp column: room for the date,
 * measured the same way; ja-JP writes it widest. Past the date the cell
 * ellipsizes, and one drag gives the rest back.
 */
export const TIMESTAMP_COLUMN_MIN_WIDTH = 140;
