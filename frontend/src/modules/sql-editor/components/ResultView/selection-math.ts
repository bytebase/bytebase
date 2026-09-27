import { sortBy } from "lodash-es";

export interface SelectionState {
  rows: number[];
  columns: number[];
}

export const isSingleCellSelected = (s: SelectionState): boolean =>
  s.rows.length === 1 && s.columns.length === 1;

export const toggleRowInSelection = (
  prev: SelectionState,
  row: number
): SelectionState => {
  const baseRows = isSingleCellSelected(prev) ? [] : prev.rows;
  return {
    rows: sortBy(
      baseRows.includes(row)
        ? baseRows.filter((r) => r !== row)
        : [...baseRows, row]
    ),
    columns: [],
  };
};

export const toggleColumnInSelection = (
  prev: SelectionState,
  column: number
): SelectionState => {
  const baseCols = isSingleCellSelected(prev) ? [] : prev.columns;
  return {
    rows: [],
    columns: sortBy(
      baseCols.includes(column)
        ? baseCols.filter((c) => c !== column)
        : [...baseCols, column]
    ),
  };
};

export const toggleCellInSelection = (
  prev: SelectionState,
  row: number,
  column: number
): SelectionState => {
  if (prev.rows.includes(row) && prev.columns.includes(column)) {
    return { rows: [], columns: [] };
  }
  return { rows: [row], columns: [column] };
};
