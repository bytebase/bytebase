// Layout for the forms HumanizeTs renders, kept out of the component so a
// test that stubs the component still has the real widths to lay out with.

/**
 * The width of a table column that holds a form on one line: the widest
 * rendering (a two-digit hour, a zone as long as GMT-3:30) at a table's 14px,
 * plus a cell's 32px of padding. Measured in every locale the app ships;
 * en-US is the widest in all three forms.
 *
 * A column may be dragged below its form only where the tooltip restores what
 * the truncation hides. Compact's tooltip is the full date-time, so it may.
 * Operational may not, because its zone has to stay visible; nor may the full
 * date-time, whose tooltip is the age and cannot give the value back.
 */
export const TIMESTAMP_COLUMN = {
  compact: { width: 190, minWidth: 120 },
  operational: { width: 260, minWidth: 260 },
  datetime: { width: 280, minWidth: 280 },
} as const;
