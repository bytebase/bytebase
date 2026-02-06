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
import type { QueryRow, RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain, isDescendantOf } from "@/utils";
import { type SQLResultViewContext } from "../../context";
import {
  type BinaryFormatContext,
  detectBinaryFormat,
  formatBinaryValue,
} from "./binary-format-store";
import type { ResultTableColumn, ResultTableRow } from "./types";

const PREVENT_DISMISS_SELECTION = "bb-prevent-dismiss-selection";

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
  copy: () => boolean;
};

export const KEY = Symbol(
  "bb.sql-editor.result-view.selection"
) as InjectionKey<SelectionContext>;

export const provideSelectionContext = ({
  rows,
  columns,
  binaryFormatContext,
  resultViewContext,
}: {
  rows: ComputedRef<ResultTableRow[]>;
  columns: ComputedRef<ResultTableColumn[]>;
  binaryFormatContext: BinaryFormatContext;
  resultViewContext: SQLResultViewContext;
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
    return resultViewContext.disallowCopyingData.value;
  });
  const isCellSelected = computed(
    () => state.value.rows.length === 1 && state.value.columns.length === 1
  );

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

  useEventListener("click", (e) => {
    if (copying.value) return;
    if (isDescendantOf(e.target as Element, `.${PREVENT_DISMISS_SELECTION}`)) {
      return;
    }
    deselect();
  });

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

  const getValues = () => {
    if (isCellSelected.value) {
      // single cell selected
      const row = rows.value[state.value.rows[0]];
      if (!row) {
        return "";
      }
      const cell = row.item.values[state.value.columns[0]];
      if (!cell) {
        return "";
      }

      // Get the cell value
      return getFormattedValue({
        value: cell,
        colIndex: state.value.columns[0],
        rowIndex: state.value.rows[0],
      });
    }

    // build column name
    let columnNames = ["index", ...columns.value.map((c) => c.name)];
    if (state.value.columns.length > 0) {
      columnNames = state.value.columns.map(
        (columnIndex) => columns.value[columnIndex]?.name ?? ""
      );
    }

    if (state.value.rows.length > 0) {
      // multi-rows selected
      const data: QueryRow[] = [];
      for (const rowIndex of state.value.rows) {
        const d = rows.value[rowIndex].item;
        if (!d) {
          continue;
        }
        data.push(d);
      }
      if (data.length === 0) {
        return "";
      }

      const value = data
        .map((row, rowIdx) => {
          return row.values
            .map((cell, colIdx) => {
              return getFormattedValue({
                value: cell,
                colIndex: colIdx,
                rowIndex: state.value.rows[rowIdx],
              });
            })
            .join("\t");
        })
        .join("\n");

      return `${columnNames.join("\t")}\n${value}`;
    }

    if (state.value.columns.length > 0) {
      // multi-columns selected
      const values: RowValue[][] = [];
      for (const row of rows.value) {
        const cells: RowValue[] = [];
        for (const column of state.value.columns) {
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
              return getFormattedValue({
                value: cell,
                rowIndex: rowIdx,
                colIndex: state.value.columns[colIdx],
              });
            })
            .join("\t")
        )
        .join("\n");

      return `${columnNames.join("\t")}\n${value}`;
    }

    return "";
  };

  const copy = () => {
    if (disabled.value || !isSupported.value) {
      return false;
    }
    const values = getValues();
    if (!values) {
      return false;
    }
    copying.value = true;
    copyTextToClipboard(values)
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
    return true;
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
      const copied = copy();
      if (copied) {
        e.preventDefault();
        e.stopImmediatePropagation();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.copied"),
        });
      }
    }
  });

  const context: SelectionContext = {
    state,
    disabled,
    toggleSelectRow,
    toggleSelectColumn,
    toggleSelectCell,
    deselect,
    copy,
  };
  provide(KEY, context);
  return context;
};

export const useSelectionContext = () => {
  return inject(KEY)!;
};
