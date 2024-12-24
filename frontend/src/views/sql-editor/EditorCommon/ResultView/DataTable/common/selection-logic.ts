import type { Cell, Table } from "@tanstack/vue-table";
import { useEventListener } from "@vueuse/core";
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
import { extractSQLRowValue, isDescendantOf, toClipboard } from "@/utils";
import { useSQLResultViewContext } from "../../context";

export const PREVENT_DISMISS_SELECTION = "bb-prevent-dismiss-selection";

export type SelectionState = {
  row: number;
  column: number;
};

export type SelectionContext = {
  state: Ref<SelectionState>;
  disabled: Ref<boolean>;
  selectRow: (row: number) => void;
  selectColumn: (column: number) => void;
  deselect: () => void;
  copy: () => void;
};

export const KEY = Symbol(
  "bb.sql-editor.result-view.selection"
) as InjectionKey<SelectionContext>;

export const provideSelectionContext = (table: Ref<Table<QueryRow>>) => {
  const { t } = useI18n();

  const copying = ref(false);
  const state = ref<SelectionState>({
    row: -1,
    column: -1,
  });
  const resultViewContext = useSQLResultViewContext();
  const disabled = computed(() => {
    return resultViewContext.disallowCopyingData.value;
  });
  const selectRow = (row: number) => {
    if (disabled.value) return;
    state.value = { row, column: -1 };
  };
  const selectColumn = (column: number) => {
    if (disabled.value) return;
    state.value = { row: -1, column };
  };
  const deselect = () => {
    state.value = {
      row: -1,
      column: -1,
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
    if (state.value.row >= 0) {
      const row = table.value.getPrePaginationRowModel().rows[state.value.row];
      if (!row) {
        return "";
      }
      const cells = row.getVisibleCells();
      return cells
        .map((cell) =>
          String(extractSQLRowValue(cell.getValue() as RowValue).plain)
        )
        .join("\t");
    } else if (state.value.column >= 0) {
      const rows = table.value.getRowModel().rows;
      const cells: Cell<QueryRow, unknown>[] = [];
      for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        const cell = row.getVisibleCells()[state.value.column];
        if (!cell) continue;
        cells.push(cell);
      }
      if (cells.length === 0) {
        return "";
      }
      return cells
        .map((cell) =>
          String(extractSQLRowValue(cell.getValue() as RowValue).plain)
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
    if (state.value.row < 0 && state.value.column < 0) return;
    if ((e.key === "c" || e.key === "C") && (e.metaKey || e.ctrlKey)) {
      const copied = copy();
      if (copied) {
        e.preventDefault();
        e.stopImmediatePropagation();
      }
    }
  });

  const context: SelectionContext = {
    state,
    disabled,
    selectRow,
    selectColumn,
    deselect,
    copy,
  };
  provide(KEY, context);
  return context;
};

export const useSelectionContext = () => {
  return inject(KEY)!;
};
