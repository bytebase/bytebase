import { useEventListener, usePointer } from "@vueuse/core";
import { reject } from "lodash-es";
import type { Ref } from "vue";
import { computed, onBeforeUnmount, reactive, watch } from "vue";
import { minmax } from "@/utils";

const MAX_AUTO_ADJUST_SCAN_ROWS = 20;

export type TableResizeOptions = {
  containerWidth: Ref<number>;
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

  const normalizeWidth = (width: number) => {
    if (state.columns.length === 1) {
      // When there is only one column, display it with full width.
      // minus 1px to avoid unexpected horizontal scrollbar.
      return options.containerWidth.value - 1;
    }
    return minmax(width, options.minWidth, options.maxWidth);
  };

  const reset = () => {
    state.drag = undefined;
    state.autoAdjusting.clear();
    state.columns = new Array(options.columnCount.value);

    const indexList: number[] = [];
    for (let i = 0; i < state.columns.length; i++) {
      indexList.push(i);
      state.columns[i] = {
        width: options.minWidth,
      };
    }

    // Calculate a friendly width for every column when the first render happened.
    const scanner = async (n = 0) => {
      // Wait for several frames for the initial render of the table
      requestAnimationFrame(() => {
        const headerTable = options.queryTableHeaderElement();
        const bodyTable = options.queryTableBodyElement();
        const trList = [
          ...Array.from(headerTable?.querySelectorAll("tr") || []),
          ...Array.from(
            bodyTable?.querySelectorAll("tr.n-data-table-tr") || []
          ),
        ] as HTMLElement[];
        if (!headerTable || !bodyTable || trList.length === 0) {
          return scanner(n + 1);
        }

        const maxAutoAdjustColumns = Math.ceil(
          options.containerWidth.value / options.minWidth
        );
        console.log("maxAutoAdjustColumns", maxAutoAdjustColumns);
        autoAdjustColumnWidth(indexList.slice(0, maxAutoAdjustColumns)).catch(
          () => {
            for (let i = 0; i < state.columns.length; i++) {
              state.columns[i].width = options.defaultWidth;
            }
          }
        );
      });
    };
    scanner();
  };

  // Automatically estimate the width of a column.
  // Like double-clicking a cell border of Excel.
  // Able to estimate multiple columns in a time.
  const autoAdjustColumnWidth = (
    indexList: number[]
  ): Promise<Map<number, number>> => {
    return new Promise((resolve) => {
      const headerTable = options.queryTableHeaderElement();
      const bodyTable = options.queryTableBodyElement();
      const trList = [
        ...Array.from(headerTable?.querySelectorAll("tr") || []),
        ...Array.from(bodyTable?.querySelectorAll("tr.n-data-table-tr") || []),
      ] as HTMLElement[];
      console.log(
        "autoAdjustColumnWidth",
        indexList.join("|"),
        `${trList.length} rows`
      );
      if (!headerTable || !bodyTable || trList.length === 0) {
        return reject(undefined);
      }

      const numScanRows = Math.min(trList.length, MAX_AUTO_ADJUST_SCAN_ROWS);
      const cellsMapByColumn = new Map<number, Array<HTMLElement>>();
      indexList.forEach((index) => {
        const cells: Array<HTMLElement> = [];
        let found = false;
        for (let row = 0; row < numScanRows; row++) {
          const tr = trList[row];
          const children = Array.from(tr.children).filter((child) => {
            return isNDataTableCell(child);
          });
          if (children.length === 0) {
            return;
          }
          const first = children[0];
          const offset = parseInt(
            first.getAttribute("data-col-key") || "0",
            10
          );
          const cell = children[index - offset];
          if (cell) {
            found = true;
            cells.push(cell);
          }
        }
        if (found) {
          cellsMapByColumn.set(index, cells);
        }
      });
      console.log(
        Array.from(cellsMapByColumn.entries())
          .map(([index, cells]) => `${index}(${cells.length})`)
          .join("|")
      );

      // Update the cells' and table's style to estimate the width.
      for (const index of cellsMapByColumn.keys()) {
        state.autoAdjusting.add(index);
      }

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

      // Read the rendered widths.
      const widthMapByColumn = new Map<number, number>();
      indexList.forEach((index) => {
        const cells = cellsMapByColumn.get(index);
        if (!cells) {
          return;
        }

        const stretchedWidths = cells.map((cell) => getElementWidth(cell));
        const finalWidth = normalizeWidth(Math.max(...stretchedWidths));
        console.log(`col#${index}`, stretchedWidths.join("|"), finalWidth);

        widthMapByColumn.set(index, finalWidth);
      });

      console.log(
        Array.from(widthMapByColumn.entries())
          .map(([index, width]) => `${index}:${width}`)
          .join("|")
      );

      indexList.forEach((index) => {
        const width = widthMapByColumn.get(index) ?? options.defaultWidth;

        const column = state.columns[index];
        if (!column) {
          // Sometimes the `columns` is out-of-sync with the `indexList`
          // so we need to detect and suppress errors here.
          // Only occurs in dev hot reload mode.
          return;
        }
        column.width = width;
      });

      // Reset the cells' and table's style.
      for (const [_index, cells] of cellsMapByColumn.entries()) {
        cells.forEach((cell) => {
          if (!cell) return;
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

      resolve(widthMapByColumn);
      // Style bindings will be automatically updated next frame.
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

  const getColumnWidth = (index: number) => {
    return state.columns[index]?.width ?? options.defaultWidth;
  };

  const totalWidth = computed(() => {
    let totalWidth = 0;
    for (let index = 0; index < options.columnCount.value; index++) {
      const columnWidth = state.autoAdjusting.has(index)
        ? options.maxWidth
        : getColumnWidth(index);
      totalWidth += columnWidth;
    }
    return totalWidth;
  });

  const getTableRenderWidth = () => {
    return Math.min(options.containerWidth.value, totalWidth.value + 2);
  };

  const getTableScrollWidth = () => {
    return totalWidth.value;
  };

  onBeforeUnmount(() => {
    toggleDragStyle(scroller, false);
  });

  return {
    reset,
    state,
    getColumnWidth,
    getTableRenderWidth,
    getTableScrollWidth,
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

const getElementWidth = (elem: HTMLElement | undefined) => {
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
    document.body.classList.add("select-none");
    // Set the cursor style.
    document.body.classList.add("cursor-col-resize");
  } else {
    requestAnimationFrame(() => {
      document.body.classList.remove("select-none");
      document.body.classList.remove("cursor-col-resize");
    });
  }
};

const isNDataTableCell = (elem: Element): elem is HTMLElement => {
  return (
    elem.classList.contains("n-data-table-th") ||
    elem.classList.contains("n-data-table-td")
  );
};
