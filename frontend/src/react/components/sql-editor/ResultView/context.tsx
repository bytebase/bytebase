import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { pushNotification } from "@/store";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { isDescendantOf } from "@/utils/dom";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import {
  type BinaryFormat,
  type BinaryFormatParams,
  detectBinaryFormat,
  formatBinaryValue,
  type GetBinaryFormatParams,
  getCellKey,
  getColumnKey,
} from "./binary-format";
import {
  isSingleCellSelected,
  type SelectionState,
  toggleCellInSelection,
  toggleColumnInSelection,
  toggleRowInSelection,
} from "./selection-math";
import type { ResultTableColumn, ResultTableRow } from "./types";

// =============================================================================
// SQLResultViewContext (dark / disallowCopyingData / detail-cell)
// =============================================================================

export interface ResultViewDetail {
  row: number;
  col: number;
}

export interface SQLResultViewContext {
  dark: boolean;
  disallowCopyingData: boolean;
  detail: ResultViewDetail | undefined;
  setDetail: (detail: ResultViewDetail | undefined) => void;
}

const SQLResultViewCtx = createContext<SQLResultViewContext | null>(null);

export function useSQLResultViewContext(): SQLResultViewContext {
  const ctx = useContext(SQLResultViewCtx);
  if (!ctx) {
    throw new Error(
      "useSQLResultViewContext must be called inside <SQLResultViewProvider>"
    );
  }
  return ctx;
}

// =============================================================================
// BinaryFormatContext (per-cell + per-column binary-format Map)
// =============================================================================

export interface BinaryFormatContext {
  getBinaryFormat: (params: GetBinaryFormatParams) => BinaryFormat | undefined;
  setBinaryFormat: (params: BinaryFormatParams) => void;
}

const BinaryFormatCtx = createContext<BinaryFormatContext | null>(null);

export function useBinaryFormatContext(): BinaryFormatContext {
  const ctx = useContext(BinaryFormatCtx);
  if (!ctx) {
    throw new Error(
      "useBinaryFormatContext must be called inside <SQLResultViewProvider>"
    );
  }
  return ctx;
}

// =============================================================================
// SelectionContext (row/column/cell selection + clipboard copy)
// =============================================================================

export type { SelectionState };

export interface SelectionContext {
  state: SelectionState;
  disabled: boolean;
  toggleSelectRow: (row: number) => void;
  toggleSelectColumn: (column: number) => void;
  toggleSelectCell: (row: number, column: number) => void;
  deselect: () => void;
  copySelected: () => void;
  copyAll: () => void;
}

const SelectionCtx = createContext<SelectionContext | null>(null);

export function useSelectionContext(): SelectionContext {
  const ctx = useContext(SelectionCtx);
  if (!ctx) {
    throw new Error(
      "useSelectionContext must be called inside <SQLResultViewProvider>"
    );
  }
  return ctx;
}

// =============================================================================
// Combined provider — mounts all three contexts in nested order.
// =============================================================================

interface SQLResultViewProviderProps {
  dark?: boolean;
  disallowCopyingData?: boolean;
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  children: ReactNode;
}

/**
 * Per-instance provider for one `<ResultView>` mount. Owns:
 *  - The dark/disallowCopying flags + the detail-cell state
 *  - The binary-format Map (per-cell and per-column overrides)
 *  - The selection state + row/column/cell toggles + clipboard copy
 *
 * Children read these via `useSQLResultViewContext`,
 * `useBinaryFormatContext`, and `useSelectionContext`.
 */
export function SQLResultViewProvider({
  dark = false,
  disallowCopyingData = false,
  rows,
  columns,
  children,
}: SQLResultViewProviderProps) {
  const { t } = useTranslation();

  // ---- SQLResultViewContext ----
  const [detail, setDetail] = useState<ResultViewDetail | undefined>(undefined);
  const sqlResultView = useMemo<SQLResultViewContext>(
    () => ({ dark, disallowCopyingData, detail, setDetail }),
    [dark, disallowCopyingData, detail]
  );

  // ---- BinaryFormatContext ----
  // Live Map kept inside state so updates re-render consumers. Mutators
  // produce a new Map reference each call so React's setState detects
  // the change.
  const [binaryFormatMap, setBinaryFormatMap] = useState<
    Map<string, BinaryFormat>
  >(() => new Map());

  const getBinaryFormat = useCallback(
    (params: GetBinaryFormatParams): BinaryFormat | undefined => {
      const { rowIndex, colIndex } = params;
      if (rowIndex !== undefined) {
        const cellKey = getCellKey({ rowIndex, colIndex });
        if (binaryFormatMap.has(cellKey)) {
          return binaryFormatMap.get(cellKey);
        }
      }
      return binaryFormatMap.get(getColumnKey({ colIndex }));
    },
    [binaryFormatMap]
  );

  const setBinaryFormat = useCallback((params: BinaryFormatParams): void => {
    const { rowIndex, colIndex, format } = params;
    const key =
      rowIndex !== undefined
        ? getCellKey({ rowIndex, colIndex })
        : getColumnKey({ colIndex });
    setBinaryFormatMap((prev) => {
      const next = new Map(prev);
      // DEFAULT means "fall back to column / auto-detect" — drop the entry
      // rather than setting "DEFAULT".
      if (format === "DEFAULT") {
        next.delete(key);
      } else {
        next.set(key, format);
      }
      return next;
    });
  }, []);

  const binaryFormat = useMemo<BinaryFormatContext>(
    () => ({ getBinaryFormat, setBinaryFormat }),
    [getBinaryFormat, setBinaryFormat]
  );

  // ---- SelectionContext ----
  const [selectionState, setSelectionState] = useState<SelectionState>({
    rows: [],
    columns: [],
  });
  const [copying, setCopying] = useState(false);
  const selectionDisabled = disallowCopyingData;

  const deselect = useCallback(() => {
    setSelectionState({ rows: [], columns: [] });
  }, []);

  const toggleSelectRow = useCallback(
    (row: number) => {
      if (selectionDisabled) return;
      setSelectionState((prev) => toggleRowInSelection(prev, row));
    },
    [selectionDisabled]
  );

  const toggleSelectColumn = useCallback(
    (column: number) => {
      if (selectionDisabled) return;
      setSelectionState((prev) => toggleColumnInSelection(prev, column));
    },
    [selectionDisabled]
  );

  const toggleSelectCell = useCallback(
    (row: number, column: number) => {
      if (selectionDisabled) return;
      setSelectionState((prev) => toggleCellInSelection(prev, row, column));
    },
    [selectionDisabled]
  );

  // ----- copy helpers -----
  const escapeTSVValue = (val: string): string => {
    if (val.includes("\t") || val.includes("\n") || val.includes('"')) {
      return `"${val.replaceAll('"', '""')}"`;
    }
    return val;
  };

  const getFormattedValue = useCallback(
    ({
      value,
      colIndex,
      rowIndex,
    }: {
      value: RowValue;
      colIndex: number;
      rowIndex: number;
    }): string => {
      // Binary cells honour any active format override, otherwise sniff
      // by content / column type.
      if (value && value.kind?.case === "bytesValue") {
        const overridden = getBinaryFormat({ colIndex, rowIndex });
        if (overridden) {
          return formatBinaryValue({
            bytesValue: value.kind.value,
            format: overridden,
          });
        }
        const detected = detectBinaryFormat({
          bytesValue: value.kind.value,
          columnType: columns[colIndex]?.columnType ?? "",
        });
        return formatBinaryValue({
          bytesValue: value.kind.value,
          format: detected,
        });
      }
      return String(extractSQLRowValuePlain(value));
    },
    [columns, getBinaryFormat]
  );

  const buildClipboardPayload = useCallback(
    (state: SelectionState): string => {
      if (isSingleCellSelected(state)) {
        const row = rows[state.rows[0]];
        if (!row) return "";
        const cell = row.item.values[state.columns[0]];
        if (!cell) return "";
        return getFormattedValue({
          value: cell,
          colIndex: state.columns[0],
          rowIndex: state.rows[0],
        });
      }

      if (state.rows.length > 0) {
        const columnNames = ["index", ...columns.map((c) => c.name)];
        const lines: string[] = [];
        for (const rowIndex of state.rows) {
          const queryRow = rows[rowIndex]?.item;
          if (!queryRow) continue;
          const cells = queryRow.values
            .map((cell, colIdx) =>
              escapeTSVValue(
                getFormattedValue({ value: cell, colIndex: colIdx, rowIndex })
              )
            )
            .join("\t");
          lines.push(`${rowIndex}\t${cells}`);
        }
        if (lines.length === 0) return "";
        return `${columnNames.join("\t")}\n${lines.join("\n")}`;
      }

      if (state.columns.length > 0) {
        const columnNames = state.columns.map((i) => columns[i]?.name ?? "");
        const lines: string[] = [];
        for (let rowIdx = 0; rowIdx < rows.length; rowIdx++) {
          const cells = state.columns.map((columnIndex, colIdx) => {
            const cell = rows[rowIdx].item.values[columnIndex];
            if (!cell) return "";
            return escapeTSVValue(
              getFormattedValue({
                value: cell,
                rowIndex: rowIdx,
                colIndex: state.columns[colIdx],
              })
            );
          });
          lines.push(cells.join("\t"));
        }
        if (lines.length === 0) return "";
        return `${columnNames.join("\t")}\n${lines.join("\n")}`;
      }

      return "";
    },
    [columns, rows, getFormattedValue]
  );

  const copyToClipboard = useCallback(
    async (state: SelectionState) => {
      if (selectionDisabled || copying) return;
      const payload = buildClipboardPayload(state);
      if (!payload) return;
      setCopying(true);
      try {
        await navigator.clipboard.writeText(payload);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
      } catch (err) {
        const errors = [t("common.failed")];
        if (err instanceof Error) errors.push(err.message);
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: errors.join(": "),
        });
      } finally {
        // Defer so we don't immediately re-permit copy via a
        // bubbled keyboard repeat.
        requestAnimationFrame(() => setCopying(false));
      }
    },
    [selectionDisabled, copying, buildClipboardPayload, t]
  );

  const copySelected = useCallback(() => {
    void copyToClipboard(selectionState);
  }, [copyToClipboard, selectionState]);

  const copyAll = useCallback(() => {
    void copyToClipboard({
      rows: rows.map((_, i) => i),
      columns: [],
    });
  }, [copyToClipboard, rows]);

  // Click outside the result-scroll buttons → deselect (mirrors Vue
  // global click listener in `selection-logic.ts`).
  useEffect(() => {
    if (copying) return;
    const handler = (e: MouseEvent) => {
      if (isDescendantOf(e.target as Element, ".result-scroll-buttons")) {
        return;
      }
      deselect();
    };
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, [copying, deselect]);

  // Esc → deselect; Cmd/Ctrl+C → copy selected.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (selectionDisabled) return;
      if (
        selectionState.columns.length === 0 &&
        selectionState.rows.length === 0
      ) {
        return;
      }
      if (e.key === "Escape") {
        deselect();
      }
      if ((e.key === "c" || e.key === "C") && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        e.stopImmediatePropagation();
        copySelected();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [selectionDisabled, selectionState, deselect, copySelected]);

  const selection = useMemo<SelectionContext>(
    () => ({
      state: selectionState,
      disabled: selectionDisabled,
      toggleSelectRow,
      toggleSelectColumn,
      toggleSelectCell,
      deselect,
      copySelected,
      copyAll,
    }),
    [
      selectionState,
      selectionDisabled,
      toggleSelectRow,
      toggleSelectColumn,
      toggleSelectCell,
      deselect,
      copySelected,
      copyAll,
    ]
  );

  return (
    <SQLResultViewCtx value={sqlResultView}>
      <BinaryFormatCtx value={binaryFormat}>
        <SelectionCtx value={selection}>{children}</SelectionCtx>
      </BinaryFormatCtx>
    </SQLResultViewCtx>
  );
}
