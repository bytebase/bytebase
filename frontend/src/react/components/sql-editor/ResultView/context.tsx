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
import { useAppStore } from "@/react/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
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
  type CopyFormatter,
  type CopyScope,
  formatAsText,
} from "./copy-formats";
import {
  type SelectionState,
  toggleCellInSelection,
  toggleColumnInSelection,
  toggleRowInSelection,
} from "./selection-math";
import type { ResultTableColumn, ResultTableRow } from "./types";

// =============================================================================
// SQLResultViewContext (disallowCopyingData / detail-cell)
// =============================================================================

export interface ResultViewDetail {
  row: number;
  col: number;
}

export interface SQLResultViewContext {
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
  // Copy the result to the clipboard. `scope` "selected" uses the current
  // selection (falling back to all rows for row-oriented formats); "all"
  // copies every row. `format` chooses the rendering — `formatAsText` (plain
  // TSV, mirrors the grid selection), `formatAsCSV`, or `formatAsSQL`.
  copy: (scope: CopyScope, format: CopyFormatter) => void;
  // Whether "copy as SQL (INSERT)" is meaningful: copying is allowed and the
  // engine has a SQL INSERT form. Callers use it to show/hide the SQL option.
  canCopyAsInsert: boolean;
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

// Non-SQL engines have no meaningful INSERT form — hide "copy as INSERT".
const NON_SQL_ENGINES = new Set<Engine>([
  Engine.MONGODB,
  Engine.REDIS,
  Engine.ELASTICSEARCH,
  Engine.COSMOSDB,
  Engine.DYNAMODB,
]);

interface SQLResultViewProviderProps {
  disallowCopyingData?: boolean;
  engine: Engine;
  // Connected schema, used to qualify the generated INSERT's table name.
  schema?: string;
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  children: ReactNode;
}

/**
 * Per-instance provider for one `<ResultView>` mount. Owns:
 *  - The disallowCopying flag + the detail-cell state
 *  - The binary-format Map (per-cell and per-column overrides)
 *  - The selection state + row/column/cell toggles + clipboard copy
 *
 * Children read these via `useSQLResultViewContext`,
 * `useBinaryFormatContext`, and `useSelectionContext`.
 */
export function SQLResultViewProvider({
  disallowCopyingData = false,
  engine,
  schema,
  rows,
  columns,
  children,
}: SQLResultViewProviderProps) {
  const { t } = useTranslation();

  // ---- SQLResultViewContext ----
  const [detail, setDetail] = useState<ResultViewDetail | undefined>(undefined);
  const sqlResultView = useMemo<SQLResultViewContext>(
    () => ({ disallowCopyingData, detail, setDetail }),
    [disallowCopyingData, detail]
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

  const writeTextToClipboard = useCallback(
    async (payload: string) => {
      if (selectionDisabled || copying) return;
      if (!payload) return;
      setCopying(true);
      try {
        await navigator.clipboard.writeText(payload);
        useAppStore.getState().notify({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
      } catch (err) {
        const errors = [t("common.failed")];
        if (err instanceof Error) errors.push(err.message);
        useAppStore.getState().notify({
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
    [selectionDisabled, copying, t]
  );

  const canCopyAsInsert = !selectionDisabled && !NON_SQL_ENGINES.has(engine);

  // Unified copy: resolve the cell-value accessor + payload context once, then
  // delegate rendering to the chosen formatter (formatAsText / CSV / SQL).
  const copy = useCallback(
    (scope: CopyScope, format: CopyFormatter) => {
      if (selectionDisabled) return;
      const payload = format({
        scope,
        selection: selectionState,
        rows,
        columns,
        engine,
        schema,
        getFormattedValue: (rowIndex, colIndex) => {
          const cell = rows[rowIndex]?.item.values[colIndex];
          return cell
            ? getFormattedValue({ value: cell, rowIndex, colIndex })
            : "";
        },
      });
      void writeTextToClipboard(payload);
    },
    [
      selectionDisabled,
      selectionState,
      rows,
      columns,
      engine,
      schema,
      getFormattedValue,
      writeTextToClipboard,
    ]
  );

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
        if (window.getSelection()?.toString()) {
          return;
        }
        e.preventDefault();
        e.stopImmediatePropagation();
        copy("selected", formatAsText);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [selectionDisabled, selectionState, deselect, copy]);

  const selection = useMemo<SelectionContext>(
    () => ({
      state: selectionState,
      disabled: selectionDisabled,
      toggleSelectRow,
      toggleSelectColumn,
      toggleSelectCell,
      deselect,
      copy,
      canCopyAsInsert,
    }),
    [
      selectionState,
      selectionDisabled,
      toggleSelectRow,
      toggleSelectColumn,
      toggleSelectCell,
      deselect,
      copy,
      canCopyAsInsert,
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
