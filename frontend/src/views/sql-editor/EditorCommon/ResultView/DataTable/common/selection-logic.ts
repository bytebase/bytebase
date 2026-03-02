import { useClipboard, useEventListener } from "@vueuse/core";
import { sortBy } from "lodash-es";
import {
  type ComputedRef,
  computed,
  type InjectionKey,
  inject,
  nextTick,
  provide,
  type Ref,
  ref,
} from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils";
import {
  type BinaryFormatContext,
  detectBinaryFormat,
  formatBinaryValue,
} from "./binary-format-store";
import type { ResultTableColumn, ResultTableRow } from "./types";

export type SelectionState = {
  rows: number[];
  columns: number[];
};

export type SelectionContext = {
  state: Ref<SelectionState>;
  disabled: Ref<boolean>;
  toggleSelectRow: (row: number) => void;
  toggleSelectColumn: (column: number) => void;
  toggleSelectCell: (row: number, column: number) => void;
  deselect: () => void;
  copySelected: () => void;
  copyAll: () => void;
};

export const KEY = Symbol(
  "bb.sql-editor.result-view.selection"
) as InjectionKey<SelectionContext>;

export const provideSelectionContext = ({
  rows,
  columns,
  binaryFormatContext,
  disallowCopyingData,
}: {
  rows: ComputedRef<ResultTableRow[]>;
  columns: ComputedRef<ResultTableColumn[]>;
  binaryFormatContext: BinaryFormatContext;
  disallowCopyingData: ComputedRef<boolean>;
}) => {
  const { t } = useI18n();
  const { getBinaryFormat } = binaryFormatContext;

  const { copy: copyTextToClipboard, isSupported } = useClipboard({
    legacy: true,
  });

  const copying = ref(false);
  const state = ref<SelectionState>({
    rows: [],
    columns: [],
  });
  const disabled = computed(() => {
    return disallowCopyingData.value;
  });
  const checkIsSingleCellSelected = (state: SelectionState) => {
    return state.rows.length === 1 && state.columns.length === 1;
  };
  const isCellSelected = computed(() => checkIsSingleCellSelected(state.value));

  const deselect = () => {
    state.value = {
      rows: [],
      columns: [],
    };
  };

  const toggleSelectRow = (row: number) => {
    if (disabled.value) return;
    if (isCellSelected.value) {
      deselect();
    }
    state.value = {
      rows: sortBy(
        state.value.rows.includes(row)
          ? state.value.rows.filter((r) => r !== row)
          : [...state.value.rows, row]
      ),
      columns: [],
    };
  };

  const toggleSelectColumn = (column: number) => {
    if (disabled.value) return;
    if (isCellSelected.value) {
      deselect();
    }
    state.value = {
      rows: [],
      columns: sortBy(
        state.value.columns.includes(column)
          ? state.value.columns.filter((c) => c !== column)
          : [...state.value.columns, column]
      ),
    };
  };

  const toggleSelectCell = (row: number, column: number) => {
    if (disabled.value) return;
    if (
      state.value.rows.includes(row) &&
      state.value.columns.includes(column)
    ) {
      deselect();
      return;
    }
    state.value = {
      rows: [row],
      columns: [column],
    };
  };

  // Escape values that contain TSV delimiters (\t, \n) or double quotes
  // by wrapping in double quotes (standard TSV/CSV quoting convention).
  const escapeTSVValue = (val: string): string => {
    if (val.includes("\t") || val.includes("\n") || val.includes('"')) {
      return `"${val.replaceAll('"', '""')}"`;
    }
    return val;
  };

  const getFormattedValue = ({
    value,
    colIndex,
    rowIndex,
  }: {
    value: RowValue;
    colIndex: number;
    rowIndex: number;
  }) => {
    // Special handling for binary data (proto-es oneof pattern)
    if (value && value.kind?.case === "bytesValue") {
      // First check if there's a column format override
      const binaryFormat = getBinaryFormat({
        colIndex,
        rowIndex,
      });
      if (binaryFormat) {
        // Column format overrides take precedence
        return formatBinaryValue({
          bytesValue: value.kind.value,
          format: binaryFormat,
        });
      }

      const detectedFormat = detectBinaryFormat({
        bytesValue: value.kind.value,
        columnType: columns.value[colIndex].columnType,
      });
      return formatBinaryValue({
        bytesValue: value.kind.value,
        format: detectedFormat,
      });
    }

    // Fall back to default formatting
    return String(extractSQLRowValuePlain(value));
  };

  const getValues = (state: SelectionState) => {
    if (checkIsSingleCellSelected(state)) {
      // single cell selected
      const row = rows.value[state.rows[0]];
      if (!row) {
        return "";
      }
      const cell = row.item.values[state.columns[0]];
      if (!cell) {
        return "";
      }

      // Get the cell value
      return getFormattedValue({
        value: cell,
        colIndex: state.columns[0],
        rowIndex: state.rows[0],
      });
    }

    if (state.rows.length > 0) {
      const columnNames = ["index", ...columns.value.map((c) => c.name)];
      // multi-rows selected
      const data: string[] = [];
      for (const rowIndex of state.rows) {
        const queryRow = rows.value[rowIndex].item;
        if (!queryRow) {
          continue;
        }

        const rowStr = queryRow.values
          .map((cell, colIdx) => {
            return escapeTSVValue(
              getFormattedValue({
                value: cell,
                colIndex: colIdx,
                rowIndex,
              })
            );
          })
          .join("\t");

        data.push(`${rowIndex}\t${rowStr}`);
      }
      if (data.length === 0) {
        return "";
      }
      return `${columnNames.join("\t")}\n${data.join("\n")}`;
    }

    if (state.columns.length > 0) {
      const columnNames = state.columns.map(
        (columnIndex) => columns.value[columnIndex]?.name ?? ""
      );
      // multi-columns selected
      const values: RowValue[][] = [];
      for (const row of rows.value) {
        const cells: RowValue[] = [];
        for (const column of state.columns) {
          const cell = row.item.values[column];
          if (!cell) continue;
          cells.push(cell);
        }
        values.push(cells);
      }
      if (values.length === 0) {
        return "";
      }
      const value = values
        .map((cells, rowIdx) =>
          cells
            .map((cell, colIdx) => {
              return escapeTSVValue(
                getFormattedValue({
                  value: cell,
                  rowIndex: rowIdx,
                  colIndex: state.columns[colIdx],
                })
              );
            })
            .join("\t")
        )
        .join("\n");

      return `${columnNames.join("\t")}\n${value}`;
    }

    return "";
  };

  const copy = (state: SelectionState) => {
    if (disabled.value || !isSupported.value || copying.value) {
      return;
    }
    const values = getValues(state);
    if (!values) {
      return;
    }
    copying.value = true;
    copyTextToClipboard(values)
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
      })
      .catch((err: unknown) => {
        const errors = [t("common.failed")];
        if (err instanceof Error) {
          errors.push(err.message);
        }
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: errors.join(": "),
        });
      })
      .finally(() => {
        nextTick(() => {
          copying.value = false;
        });
      });
  };

  const copyAll = () => {
    copy({
      rows: rows.value.map((_, i) => i),
      columns: [],
    });
  };

  const copySelected = () => {
    copy(state.value);
  };

  useEventListener("keydown", (e) => {
    if (disabled.value) return;
    if (state.value.columns.length === 0 && state.value.rows.length === 0)
      return;
    // Deselect all when escape is pressed.
    if (e.key === "Escape") {
      deselect();
    }
    // Copy when Cmd/Ctrl + C is pressed.
    if ((e.key === "c" || e.key === "C") && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.stopImmediatePropagation();
      copySelected();
    }
  });

  const context: SelectionContext = {
    state,
    disabled,
    toggleSelectRow,
    toggleSelectColumn,
    toggleSelectCell,
    deselect,
    copySelected,
    copyAll,
  };
  provide(KEY, context);
  return context;
};

export const useSelectionContext = () => {
  return inject(KEY)!;
};
