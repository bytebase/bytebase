import { useEventListener, usePointer } from "@vueuse/core";
import { sumBy } from "lodash-es";
import type { Ref } from "vue";
import { computed, onBeforeUnmount, reactive, watch } from "vue";

export type TableResizeOptions = {
  tableRef: Ref<HTMLTableElement | undefined>;
  scrollerRef: Ref<HTMLElement | null | undefined>;
  minWidth: number;
  maxWidth: number;
};

type ColumnProps = {
  width: number;
};

type DragState = {
  start: number;
  index: number;
  initialWidth: number;
};

type LocalState = {
  columns: ColumnProps[];
  drag?: DragState;
  isAutoAdjusting: boolean;
};

const useTableResize = (options: TableResizeOptions) => {
  const state = reactive<LocalState>({
    columns: [],
    drag: undefined,
    isAutoAdjusting: false,
  });
  const pointer = usePointer();

  const table = computed(() => options.tableRef.value!);

  const containerWidth = computed(() => {
    return options.scrollerRef?.value?.scrollWidth || 0;
  });

  const normalizeWidth = (width: number) => {
    if (state.columns.length === 1) {
      // When there is only one column, display it with full width.
      // minus 1px to avoid unexpected horizontal scrollbar.
      return containerWidth.value - 1;
    }
    const { maxWidth, minWidth } = options;
    if (width > maxWidth) return maxWidth;
    if (width < minWidth) return minWidth;
    return width;
  };

  const reset = () => {
    if (!table.value) return;

    state.drag = undefined;
    state.isAutoAdjusting = false;

    const thList = Array.from(table.value.querySelectorAll("th"));
    const columnCount = thList.length;
    state.columns = new Array(columnCount);

    const indexList: number[] = [];
    for (let i = 0; i < columnCount; i++) {
      indexList.push(i);
      state.columns[i] = {
        width: 0, // For auto adjust below
      };
    }

    // Calculate a friendly width for every column when the first render happened.
    requestAnimationFrame(() => {
      autoAdjustColumnWidth(indexList);
    });
  };

  // Automatically estimate the width of a column.
  // Like double-clicking a cell border of Excel.
  // Able to estimate multiple columns in a time.
  const autoAdjustColumnWidth = (indexList: number[]): Promise<number[]> => {
    return new Promise((resolve) => {
      const cellListOfEachColumn = indexList.map((index) => {
        const pseudo = `:nth-child(${index + 1})`;
        // Find all cells in this column
        const cellList = Array.from(
          table.value.querySelectorAll(`th${pseudo}, td${pseudo}`)
        ) as HTMLElement[];
        return cellList;
      });

      // Update the cells' and table's style to estimate the width.
      state.isAutoAdjusting = true;
      cellListOfEachColumn.forEach((cellList) => {
        cellList.forEach((cell) => {
          cell.style.whiteSpace = "nowrap";
          cell.style.overflow = "visible";
          cell.style.width = "auto";
          cell.style.maxWidth = `${options.maxWidth}px`;
          cell.style.minWidth = `${options.minWidth}px`;
        });
      });
      const tableWidthBackup = table.value.style.width;
      table.value.style.width = "auto";

      // Wait for the next render frame.
      requestAnimationFrame(() => {
        // Read the rendered widths.

        const widthList = indexList.map((index, i) => {
          const cellList = cellListOfEachColumn[i];
          const th = cellList[0];
          const stretchedWidth = getElementWidth(th);
          const finalWidth = normalizeWidth(stretchedWidth);

          const column = state.columns[index];
          if (column) {
            // Sometimes the `columns` is out-of-sync with the `indexList`
            // so we need to detect and suppress errors here.
            // Only occurs in dev hot reload mode.
            column.width = finalWidth;
          }

          return finalWidth;
        });

        // Reset the cells' and table's style.
        cellListOfEachColumn.forEach((cellList) => {
          cellList.forEach((cell) => {
            cell.style.whiteSpace = "";
            cell.style.overflow = "";
            cell.style.width = "";
            cell.style.maxWidth = "";
            cell.style.minWidth = "";
          });
        });
        table.value.style.width = tableWidthBackup;
        state.isAutoAdjusting = false;

        resolve(widthList);
        // Style bindings will be automatically updated next frame.
      });
    });
  };

  // Record the initial state of dragging.
  const startResizing = (index: number) => {
    state.drag = {
      start: pointer.x.value,
      index,
      initialWidth: state.columns[index].width,
    };
    toggleDragStyle(table, true);
  };

  // Watch the drag moving.
  watch(pointer.x, () => {
    const { drag } = state;
    if (!drag) return;
    const { start, index, initialWidth } = drag;

    // Calculate the expectedWidth according to the pointer's position.
    const offset = pointer.x.value - start;
    const expectedWidth = initialWidth + offset;
    state.columns[index].width = normalizeWidth(expectedWidth);
    if (index === state.columns.length - 1) {
      // When resizing the last column, Keep the horizontal scroll at the end of the container.
      scrollMaxX(options.scrollerRef.value);
    }
  });

  // Reset dragging state when mouse button released.
  useEventListener("pointerup", () => {
    if (state.drag) {
      state.drag = undefined;
      toggleDragStyle(table, false);
    }
  });

  const getColumnProps = (index: number) => {
    const column = state.columns[index];
    if (!column) return {};
    return {
      style: {
        width: state.isAutoAdjusting ? "auto" : `${column.width}px`,
      },
      class: state.isAutoAdjusting ? "" : "truncate",
      "data-index": index,
    };
  };

  const getTableProps = () => {
    const totalWidth = sumBy(state.columns, (col) => col.width);
    return {
      style: {
        width: state.isAutoAdjusting ? "auto" : `${totalWidth}px`,
      },
    };
  };

  onBeforeUnmount(() => {
    toggleDragStyle(table, false);
  });

  return {
    reset,
    getColumnProps,
    getTableProps,
    autoAdjustColumnWidth,
    startResizing,
  };
};

export default useTableResize;

const getElementWidth = (elem: HTMLElement) => {
  if (!elem) return 0;
  const rect = elem.getBoundingClientRect();
  return rect.width;
};

const scrollMaxX = (elem: HTMLElement | null | undefined) => {
  if (!elem) return;
  const max = elem.scrollWidth;
  elem.scrollTo(max, elem.scrollTop);
};

const toggleDragStyle = (
  table: Ref<HTMLTableElement | undefined>,
  on: boolean
) => {
  if (on) {
    // Prevent text selection while dragging.
    table.value?.classList.add("select-none");
    // Set the cursor style.
    document.body.classList.add("cursor-col-resize");
  } else {
    table.value?.classList.remove("select-none");
    document.body.classList.remove("cursor-col-resize");
  }
};
