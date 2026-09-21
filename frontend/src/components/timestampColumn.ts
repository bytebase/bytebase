// Layout for the forms HumanizeTs renders, kept out of the component so a
// test that stubs the component still has the real widths to lay out with.

/**
 * The default width of a table column that holds a form on one line: the
 * widest rendering (a two-digit hour, a zone as long as GMT-3:30) at a
 * table's 14px, plus a cell's 32px of padding. Measured in every locale the
 * app ships; en-US is the widest in all three forms. The default shows the
 * whole value, so nothing is cut unless the reader chooses it.
 *
 * The minimum is one floor for every form -- room for the date -- rather than
 * the form's width. A reader who narrows a column past the seconds or the
 * zone has traded them for another column's room, and one drag undoes it;
 * a floor at the widest case would take that choice away from everyone to
 * serve a half-hour zone.
 */
const MIN_WIDTH = 120;

export const TIMESTAMP_COLUMN = {
  compact: { width: 190, minWidth: MIN_WIDTH },
  operational: { width: 260, minWidth: MIN_WIDTH },
  datetime: { width: 280, minWidth: MIN_WIDTH },
} as const;
