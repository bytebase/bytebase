import { describe, expect, it } from "vitest";
import {
  isSingleCellSelected,
  type SelectionState,
  toggleCellInSelection,
  toggleColumnInSelection,
  toggleRowInSelection,
} from "./selection-math";

const empty: SelectionState = { rows: [], columns: [] };

describe("isSingleCellSelected", () => {
  it("is true only for exactly one row × one column", () => {
    expect(isSingleCellSelected(empty)).toBe(false);
    expect(isSingleCellSelected({ rows: [3], columns: [1] })).toBe(true);
    expect(isSingleCellSelected({ rows: [3], columns: [] })).toBe(false);
    expect(isSingleCellSelected({ rows: [], columns: [1] })).toBe(false);
    expect(isSingleCellSelected({ rows: [3, 4], columns: [1] })).toBe(false);
    expect(isSingleCellSelected({ rows: [3], columns: [1, 2] })).toBe(false);
  });
});

describe("toggleRowInSelection", () => {
  it("adds a new row in sorted order", () => {
    expect(toggleRowInSelection({ rows: [1, 4], columns: [] }, 2)).toEqual({
      rows: [1, 2, 4],
      columns: [],
    });
  });

  it("removes an already-selected row", () => {
    expect(toggleRowInSelection({ rows: [1, 2, 4], columns: [] }, 2)).toEqual({
      rows: [1, 4],
      columns: [],
    });
  });

  it("clears columns when selecting rows (mode switch)", () => {
    expect(toggleRowInSelection({ rows: [], columns: [0, 3] }, 1)).toEqual({
      rows: [1],
      columns: [],
    });
  });

  it("treats prior single-cell selection as a fresh start", () => {
    // Prior: cell (3,1). Toggling row 5 should NOT keep row 3.
    expect(toggleRowInSelection({ rows: [3], columns: [1] }, 5)).toEqual({
      rows: [5],
      columns: [],
    });
  });

  it("preserves multi-row selection when extending", () => {
    expect(toggleRowInSelection({ rows: [1, 2], columns: [] }, 5)).toEqual({
      rows: [1, 2, 5],
      columns: [],
    });
  });
});

describe("toggleColumnInSelection", () => {
  it("adds a new column in sorted order", () => {
    expect(toggleColumnInSelection({ rows: [], columns: [0, 3] }, 2)).toEqual({
      rows: [],
      columns: [0, 2, 3],
    });
  });

  it("removes an already-selected column", () => {
    expect(
      toggleColumnInSelection({ rows: [], columns: [0, 2, 3] }, 2)
    ).toEqual({
      rows: [],
      columns: [0, 3],
    });
  });

  it("clears rows when selecting columns (mode switch)", () => {
    expect(toggleColumnInSelection({ rows: [1, 2], columns: [] }, 0)).toEqual({
      rows: [],
      columns: [0],
    });
  });

  it("treats prior single-cell selection as a fresh start", () => {
    expect(toggleColumnInSelection({ rows: [3], columns: [1] }, 4)).toEqual({
      rows: [],
      columns: [4],
    });
  });
});

describe("toggleCellInSelection", () => {
  it("selects a cell from empty", () => {
    expect(toggleCellInSelection(empty, 3, 1)).toEqual({
      rows: [3],
      columns: [1],
    });
  });

  it("toggles off when clicking the same cell", () => {
    expect(toggleCellInSelection({ rows: [3], columns: [1] }, 3, 1)).toEqual(
      empty
    );
  });

  it("replaces a different cell selection", () => {
    expect(toggleCellInSelection({ rows: [3], columns: [1] }, 5, 2)).toEqual({
      rows: [5],
      columns: [2],
    });
  });

  it("replaces a row-mode selection with a single cell", () => {
    expect(
      toggleCellInSelection({ rows: [1, 2, 4], columns: [] }, 3, 0)
    ).toEqual({
      rows: [3],
      columns: [0],
    });
  });

  it("toggles off when clicking inside a multi-row × multi-column rectangle", () => {
    // Prior multi-row + multi-column: (1,3) × (0,2) intersection includes (1,0).
    // Clicking that intersection should toggle off (current logic uses
    // membership in BOTH lists).
    expect(
      toggleCellInSelection({ rows: [1, 3], columns: [0, 2] }, 1, 0)
    ).toEqual(empty);
  });
});
