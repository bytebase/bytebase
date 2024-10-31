import type { Header, Table } from "@tanstack/vue-table";
import { useEventListener, usePointer } from "@vueuse/core";
import { reject } from "lodash-es";
import stringWidth from "string-width";
import type { MaybeRef, Ref } from "vue";
import { computed, onBeforeUnmount, reactive, unref, watch } from "vue";
import type { QueryRow, RowValue } from "@/types/proto/v1/sql_service";
import { extractSQLRowValue, minmax, wrapRefAsPromise } from "@/utils";

const MAX_AUTO_ADJUST_ROW_COUNT = 20;
const MAX_AUTO_ADJUST_COL_COUNT = 50;
const MAX_SCAN_RETRIES = 60;

export type TableResizeOptions = {
  table: MaybeRef<Table<QueryRow>>;
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
  const layoutReady = wrapRefAsPromise(
    computed(() => options.containerWidth.value > 0),
    true
  );

  const normalizeWidth = (width: number) => {
    if (state.columns.length === 1) {
      // When there is only one column, display it with full width.
      // minus 2px to avoid unexpected horizontal scrollbar.
      return options.containerWidth.value - 2;
    }
    return minmax(width, options.minWidth, options.maxWidth);
  };

  const maxAutoAdjustColCount = computed(() => {
    return Math.min(
      options.columnCount.value,
      Math.ceil(options.containerWidth.value / options.minWidth),
      MAX_AUTO_ADJUST_COL_COUNT
    );
  });
  const maxAutoAdjustRowCount = computed(() => {
    return Math.min(
      unref(options.table).getRowCount(),
      MAX_AUTO_ADJUST_ROW_COUNT
    );
  });

  const reset = () => {
    layoutReady.then(() => {
      // reset state
      state.drag = undefined;
      state.autoAdjusting.clear();
      state.columns = new Array(options.columnCount.value);

      // set initial column widths
      const indexList: number[] = [];
      for (let index = 0; index < options.columnCount.value; index++) {
        if (index < maxAutoAdjustColCount.value) {
          // for columns that will be rendered in the first screen
          // set their widths to minWidth for further render-based auto-width
          state.columns[index] = {
            width: options.minWidth,
          };
          indexList.push(index);
        } else {
          // for columns that won't be rendered in the first screen
          // guess their widths by characters
          state.columns[index] = {
            width: guessColumnWidth(index),
          };
        }
      }

      // Calculate a friendly width for every column when the first render happened.
      const scanner = async (n = 0) => {
        if (n > MAX_SCAN_RETRIES) {
          console.warn(
            "[useTableResize] MAX_SCAN_RETRIES exceeded",
            MAX_SCAN_RETRIES
          );
          return;
        }
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
          if (trList.length === 0) {
            return scanner(n + 1);
          }
          autoAdjustColumnWidth(indexList).catch(() => {
            // auto adjust failed, fallback to guessing column width.
            for (let index = 0; index < state.columns.length; index++) {
              state.columns[index].width = guessColumnWidth(index);
            }
          });
        });
      };
      scanner();
    });
  };

  // Automatically estimate the width of a column.
  // Like double-clicking a cell border of Excel.
  // Able to estimate multiple columns in a time.
  const autoAdjustColumnWidth = (
    indexList: number[]
  ): Promise<Map<number, number>> => {
    return new Promise((resolve) => {
      if (indexList.length === 0) {
        return resolve(new Map());
      }

      const headerTable = options.queryTableHeaderElement();
      const bodyTable = options.queryTableBodyElement();
      const trList = [
        ...Array.from(headerTable?.querySelectorAll("tr") || []),
        ...Array.from(
          bodyTable?.querySelectorAll("tr.n-data-table-tr") || []
        ).slice(0, maxAutoAdjustRowCount.value), // Do not adjust too many rows
      ] as HTMLElement[];
      console.debug(
        "[autoAdjustColumnWidth]",
        indexList.join("|"),
        `(${trList.length} rows)`
      );
      if (trList.length === 0) {
        return reject(undefined);
      }

      const cellsMapByColumn = new Map<number, Array<HTMLElement>>();
      indexList.forEach((index) => {
        const cells: Array<HTMLElement> = [];
        for (let row = 0; row < trList.length; row++) {
          const tr = trList[row];
          const children = Array.from(tr.children).filter((child) => {
            return isNDataTableCell(child);
          });
          if (children.length === 0) {
            return;
          }
          // When the table is scrolled horizontally, some left columns will not
          // be actually rendered.
          // So we need to find out the col index of the first rendered column
          // to get the column with offset.
          const first = children[0];
          const offset = parseInt(
            first.getAttribute("data-col-key") || "0",
            10
          );
          const cell = children[index - offset];
          if (cell) {
            cells.push(cell);
          }
        }
        if (cells.length > 0) {
          cellsMapByColumn.set(index, cells);
        }
      });
      console.debug(
        "[autoAdjustColumnWidth] found cells",
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
      const headerTableWidthBackup = headerTable?.style.width ?? "";
      const bodyTableWidthBackup = bodyTable?.style.width ?? "";
      if (headerTable) headerTable.style.width = "auto";
      if (bodyTable) bodyTable.style.width = "auto";

      // Read the rendered widths.
      const widthMapByColumn = new Map<number, number>();
      indexList.forEach((index) => {
        const cells = cellsMapByColumn.get(index);
        if (!cells) {
          return;
        }

        const stretchedWidths = cells.map((cell) => getElementWidth(cell));
        const finalWidth = normalizeWidth(Math.max(...stretchedWidths));
        console.debug(`col#${index}`, stretchedWidths.join("|"), finalWidth);

        widthMapByColumn.set(index, finalWidth);
      });

      console.debug(
        Array.from(widthMapByColumn.entries())
          .map(([index, width]) => `${index}:${width}`)
          .join("|")
      );

      indexList.forEach((index) => {
        const width = widthMapByColumn.get(index) ?? guessColumnWidth(index);

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
      if (headerTable) {
        headerTable.style.width = headerTableWidthBackup;
      }
      if (bodyTable) {
        bodyTable.style.width = bodyTableWidthBackup;
      }
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

  const guessColumnWidth = (
    index: number,
    opts = {
      em: 8,
      padding: 8,
    }
  ) => {
    const table = unref(options.table);
    const maxAutoAdjustRowCount = Math.min(
      table.getRowCount(),
      MAX_AUTO_ADJUST_ROW_COUNT
    );

    const charsWidth = (content: string) => {
      return stringWidth(content) * opts.em + opts.padding * 2;
    };

    const widths: number[] = [];

    const header = table.getFlatHeaders()[index] as Header<QueryRow, RowValue>;
    if (header) {
      const content = String(header.column.columnDef.header);
      const width = charsWidth(content) + 4 + 16; // plus sort icon and gap
      widths.push(width);
    }

    for (let r = 0; r < maxAutoAdjustRowCount; r++) {
      const row = table.getRowModel().rows[r];
      const cell = row?.getVisibleCells()[index];
      if (!cell) continue;
      const value = cell.getValue() as RowValue;
      const content = String(extractSQLRowValue(value).plain);
      widths.push(charsWidth(content));
    }

    const guessed = Math.max(...widths);
    return normalizeWidth(guessed);
  };

  onBeforeUnmount(() => {
    toggleDragStyle(scroller, false);
  });

  return {
    state,
    layoutReady,
    reset,
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
