import type { Cell, Row, Table } from "@tanstack/vue-table";
import { useEventListener } from "@vueuse/core";
import { sortBy } from "lodash-es";
import {
  computed,
  inject,
  nextTick,
  provide,
  ref,
  watch,
  type InjectionKey,
  type Ref,
} from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValuePlain, isDescendantOf, toClipboard } from "@/utils";
import { useSQLResultViewContext } from "../../context";

export const PREVENT_DISMISS_SELECTION = "bb-prevent-dismiss-selection";

export type SelectionState = {
  rows: number[];
  columns: number[];
};

export type SelectionContext = {
  state: Ref<SelectionState>;
  disabled: Ref<boolean>;
  selectRow: (row: number) => void;
  selectColumn: (column: number) => void;
  selectCell: (row: number, column: number) => void;
  deselect: () => void;
  copy: () => boolean;
};

export const KEY = Symbol(
  "bb.sql-editor.result-view.selection"
) as InjectionKey<SelectionContext>;

export const provideSelectionContext = (table: Ref<Table<QueryRow>>) => {
  const { t } = useI18n();

  const copying = ref(false);
  const state = ref<SelectionState>({
    rows: [],
    columns: [],
  });
  const resultViewContext = useSQLResultViewContext();
  const disabled = computed(() => {
    return resultViewContext.disallowCopyingData.value;
  });
  const isCellSelected = computed(
    () => state.value.rows.length === 1 && state.value.columns.length === 1
  );
  const selectRow = (row: number) => {
    if (disabled.value) return;
    if (isCellSelected.value) {
      state.value = {
        rows: [row],
        columns: [],
      };
      return;
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
  const selectColumn = (column: number) => {
    if (disabled.value) return;
    if (isCellSelected.value) {
      state.value = {
        rows: [],
        columns: [column],
      };
      return;
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
  const selectCell = (row: number, column: number) => {
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
  const deselect = () => {
    state.value = {
      rows: [],
      columns: [],
    };
  };

  useEventListener("click", (e) => {
    if (copying.value) return;
    if (isDescendantOf(e.target as Element, `.${PREVENT_DISMISS_SELECTION}`)) {
      return;
    }
    deselect();
  });

  watch(table, deselect, {
    immediate: true,
    deep: false,
  });

  const getValues = () => {
    if (state.value.rows.length === 1 && state.value.columns.length === 1) {
      const row =
        table.value.getPrePaginationRowModel().rows[state.value.rows[0]];
      if (!row) {
        return "";
      }
      const cell = row.getVisibleCells()[state.value.columns[0]];
      if (!cell) {
        return "";
      }
      return String(extractSQLRowValuePlain(cell.getValue() as RowValue));
    } else if (state.value.rows.length > 0) {
      const rows: Row<QueryRow>[] = [];
      for (const row of state.value.rows) {
        const d = table.value.getPrePaginationRowModel().rows[row];
        if (!d) {
          continue;
        }
        rows.push(d);
      }
      if (rows.length === 0) {
        return "";
      }
      return rows
        .map((row) => {
          const cells = row.getVisibleCells();
          return cells
            .map((cell) =>
              String(extractSQLRowValuePlain(cell.getValue() as RowValue))
            )
            .join("\t");
        })
        .join("\n");
    } else if (state.value.columns.length > 0) {
      const rows = table.value.getRowModel().rows;
      const values: Cell<QueryRow, unknown>[][] = [];
      for (const row of rows) {
        const cells: Cell<QueryRow, unknown>[] = [];
        for (const column of state.value.columns) {
          const cell = row.getVisibleCells()[column];
          if (!cell) continue;
          cells.push(cell);
        }
        values.push(cells);
      }
      if (values.length === 0) {
        return "";
      }
      return values
        .map((cells) =>
          cells
            .map((cell) =>
              String(extractSQLRowValuePlain(cell.getValue() as RowValue))
            )
            .join("\t")
        )
        .join("\n");
    }

    return "";
  };

  const copy = () => {
    if (disabled.value) {
      return false;
    }
    const values = getValues();
    if (!values) {
      return false;
    }
    copying.value = true;
    toClipboard(values)
      .catch((err: any) => {
        const errors = [t("common.failed")];
        if (err && err instanceof Error) {
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
    selectRow,
    selectColumn,
    selectCell,
    deselect,
    copy,
  };
  provide(KEY, context);
  return context;
};

export const useSelectionContext = () => {
  return inject(KEY)!;
};
