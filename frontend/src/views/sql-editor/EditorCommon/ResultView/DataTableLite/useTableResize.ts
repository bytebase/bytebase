import { useEventListener, usePointer } from "@vueuse/core";
import { reject, sumBy } from "lodash-es";
import type { Ref } from "vue";
import { computed, onBeforeUnmount, reactive, unref, watch } from "vue";
import { minmax } from "@/utils";

const MAX_AUTO_ADJUST_SCAN_ROWS = 20;
const MAX_AUTO_ADJUST_COLUMNS = 20;

export type TableResizeOptions = {
  scrollerRef: Ref<HTMLElement | undefined>;
  queryTableHeaderElement: () => HTMLElement | undefined;
  queryTableBodyElement: () => HTMLElement | undefined;
  columnCount: Ref<number>;
  defaultWidth: number;
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
  autoAdjusting: Set<number>;
  drag?: DragState;
};

const useTableResize = (options: TableResizeOptions) => {
  const state = reactive<LocalState>({
    columns: [],
    autoAdjusting: new Set(),
    drag: undefined,
  });
  const pointer = usePointer();
  const scroller = options.scrollerRef;

  const containerWidth = computed(() => {
    return scroller.value?.scrollWidth || 0;
  });

  const normalizeWidth = (width: number) => {
    if (state.columns.length === 1) {
      // When there is only one column, display it with full width.
      // minus 1px to avoid unexpected horizontal scrollbar.
      return containerWidth.value - 1;
    }
    return minmax(width, options.minWidth, options.maxWidth);
  };

  const reset = () => {
    state.drag = undefined;

    state.autoAdjusting.clear();

    const columnCount = unref(options.columnCount);
    state.columns = new Array(columnCount);

    const indexList: number[] = [];
    for (let i = 0; i < columnCount; i++) {
      indexList.push(i);
      state.columns[i] = {
        width: i < MAX_AUTO_ADJUST_COLUMNS ? 0 : options.defaultWidth, // For auto adjust below
      };
    }

    // Calculate a friendly width for every column when the first render happened.
    const scanner = async (n = 0) => {
      // Wait for several frames for the initial render of the table
      requestAnimationFrame(() => {
        const headerTable = options.queryTableHeaderElement();
        const bodyTable = options.queryTableBodyElement();
        if (headerTable && bodyTable) {
          autoAdjustColumnWidth(
            indexList.slice(0, MAX_AUTO_ADJUST_COLUMNS)
          ).catch(() => {
            for (let i = 0; i < state.columns.length; i++) {
              state.columns[i].width = options.defaultWidth;
            }
          });
          return;
        }
        scanner(n + 1);
      });
    };
    scanner();
  };

  // Automatically estimate the width of a column.
  // Like double-clicking a cell border of Excel.
  // Able to estimate multiple columns in a time.
  const autoAdjustColumnWidth = (indexList: number[]): Promise<number[]> => {
    return new Promise((resolve) => {
      const headerTable = options.queryTableHeaderElement();
      const bodyTable = options.queryTableBodyElement();
      const trList = [
        ...Array.from(headerTable?.querySelectorAll("tr") || []),
        ...Array.from(bodyTable?.querySelectorAll("tr") || []),
      ] as HTMLElement[];
      if (!headerTable || !bodyTable || trList.length === 0) {
        return reject(undefined);
      }

      const numScanRows = Math.min(trList.length, MAX_AUTO_ADJUST_SCAN_ROWS);
      const cellsMapByColumn = new Map<number, HTMLElement[]>();
      indexList.forEach((index) => {
        const cells: HTMLElement[] = [];
        for (let row = 0; row < numScanRows; row++) {
          const tr = trList[row];
          const cell = tr.children[index] as HTMLElement;
          if (cell && ["th", "td"].includes(cell.tagName.toLowerCase())) {
            cells.push(cell);
          }
        }
        cellsMapByColumn.set(index, cells);
        return cells;
      });

      // Update the cells' and table's style to estimate the width.
      indexList.forEach((colIndex) => state.autoAdjusting.add(colIndex));
      for (const [_index, cells] of cellsMapByColumn.entries()) {
        cells.forEach((cell) => {
          cell.style.whiteSpace = "nowrap";
          cell.style.overflow = "visible";
          cell.style.width = "auto";
          cell.style.maxWidth = `${options.maxWidth}px`;
          cell.style.minWidth = `${options.minWidth}px`;
        });
      }
      const headerTableWidthBackup = headerTable.style.width;
      const bodyTableWidthBackup = bodyTable.style.width;
      headerTable.style.width = "auto";
      bodyTable.style.width = "auto";

      // Wait for the next render frame.
      requestAnimationFrame(() => {
        // Read the rendered widths.
        const widthList = indexList.map((index) => {
          const cells = cellsMapByColumn.get(index) ?? [];
          const stretchedWidths = cells.map((cell) => getElementWidth(cell));
          const finalWidth = normalizeWidth(Math.max(...stretchedWidths));

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
        for (const [_index, cells] of cellsMapByColumn.entries()) {
          cells.forEach((cell) => {
            cell.style.whiteSpace = "";
            cell.style.overflow = "";
            cell.style.width = "";
            cell.style.maxWidth = "";
            cell.style.minWidth = "";
          });
        }
        headerTable.style.width = headerTableWidthBackup;
        bodyTable.style.width = bodyTableWidthBackup;
        state.autoAdjusting.clear();

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
    toggleDragStyle(scroller, true);
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
      scrollMaxX(scroller.value);
    }
  });

  // Reset dragging state when mouse button released.
  useEventListener("pointerup", () => {
    if (state.drag) {
      state.drag = undefined;
      toggleDragStyle(scroller, false);
    }
  });

  const getCellProps = (index: number) => {
    const column = state.columns[index];
    if (!column) return {};
    return {
      style: {
        width: `${column.width}px`,
      },
      class: "truncate",
      "data-index": index,
    };
  };

  const getTableProps = () => {
    const width =
      state.autoAdjusting.size > 0
        ? "auto"
        : `calc(min(100%, ${sumBy(state.columns, (col) => col.width) + 2}px))`;
    return {
      style: {
        width,
      },
    };
  };

  onBeforeUnmount(() => {
    toggleDragStyle(scroller, false);
  });

  return {
    reset,
    state,
    getCellProps,
    getTableProps,
    startResizing,
    autoAdjustColumnWidth,
  };
};

export default useTableResize;

const scrollMaxX = (elem: HTMLElement | undefined) => {
  if (!elem) return;
  const max = elem.scrollWidth;
  elem.scrollTo(max, elem.scrollTop);
};

const getElementWidth = (elem: HTMLElement) => {
  if (!elem) return 0;
  const rect = elem.getBoundingClientRect();
  return rect.width;
};

const toggleDragStyle = (
  element: Ref<HTMLElement | undefined>,
  on: boolean
) => {
  if (on) {
    // Prevent text selection while dragging.
    element.value?.classList.add("select-none");
    // Set the cursor style.
    document.body.classList.add("cursor-col-resize");
  } else {
    requestAnimationFrame(() => {
      element.value?.classList.remove("select-none");
      document.body.classList.remove("cursor-col-resize");
    });
  }
};
