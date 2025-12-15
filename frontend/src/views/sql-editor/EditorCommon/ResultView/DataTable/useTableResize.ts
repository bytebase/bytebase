import { useEventListener, usePointer } from "@vueuse/core";
import { sumBy } from "lodash-es";
import type { Ref } from "vue";
import { computed, nextTick, onBeforeUnmount, reactive, watch } from "vue";

export type TableResizeOptions = {
  tableRef: Ref<HTMLTableElement | undefined>;
  containerRef: Ref<HTMLElement | null | undefined>;
  minWidth: number;
  maxWidth: number;
  // Optional: provide row cell content for minimum width calculation
  // Returns the display text for a given column index (0-based, excluding the index column)
  getRowCellContent?: (columnIndex: number) => string | undefined;
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

  const table = computed(() => options.tableRef.value);

  const containerWidth = computed(() => {
    return options.containerRef?.value?.scrollWidth || 0;
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
    // Use auto layout to calculate the width of each column.
    state.isAutoAdjusting = true;

    nextTick(() => {
      if (!table.value) return;

      const thList = Array.from(table.value.querySelectorAll("th"));

      // Font settings for measuring cell text width
      // Cell font: text-sm (14px) monospace as defined in TableCell.vue
      const cellFont = "14px ui-monospace, monospace";
      // Cell padding: px-1 = 0.25rem * 2 = 8px
      const cellPadding = 8;

      const baseWidths = thList.map((th, index) => {
        // Get header content width from the first child element (the content div)
        // This includes text + icons and avoids inflation from auto-layout
        const content = th.firstElementChild as HTMLElement | null;
        const thStyle = getComputedStyle(th);
        const paddingLeft = parseFloat(thStyle.paddingLeft) || 0;
        const paddingRight = parseFloat(thStyle.paddingRight) || 0;
        const borderLeft = parseFloat(thStyle.borderLeftWidth) || 0;
        const borderRight = parseFloat(thStyle.borderRightWidth) || 0;

        // For the first column (index column), use scrollWidth to get exact content width
        if (index === 0) {
          if (content) {
            const contentStyle = getComputedStyle(content);
            const marginLeft = parseFloat(contentStyle.marginLeft) || 0;
            const marginRight = parseFloat(contentStyle.marginRight) || 0;
            return (
              content.scrollWidth +
              borderLeft +
              borderRight +
              marginLeft +
              marginRight
            );
          }
        }

        // For data columns, measure header content scrollWidth (includes text + icons)
        let headerWidth = options.minWidth;
        if (content) {
          headerWidth =
            content.scrollWidth +
            paddingLeft +
            paddingRight +
            borderLeft +
            borderRight;
        }

        // Also consider first row content width if available
        // index-1 because the first column (index 0) is the row index column
        if (options.getRowCellContent && index > 0) {
          const cellContent = options.getRowCellContent(index - 1);
          // Only use cell content if it's non-empty
          if (
            cellContent !== undefined &&
            cellContent !== null &&
            cellContent.length > 0
          ) {
            const textWidth = measureTextWidth(String(cellContent), cellFont);
            const cellWidth = textWidth + cellPadding;
            // Use the larger of header width and cell content width
            headerWidth = Math.max(headerWidth, cellWidth);
          }
        }

        // Apply min/max constraints
        return Math.min(
          options.maxWidth,
          Math.max(options.minWidth, headerWidth)
        );
      });

      state.columns = baseWidths.map((width) => ({ width }));
      // After calculating the width, use fixed layout to keep the width stable.
      state.isAutoAdjusting = false;
    });
  };

  // Record the initial state of dragging.
  const startResizing = (index: number) => {
    // Use actual DOM width as the initial width to ensure accuracy
    const th = table.value?.querySelectorAll("th")[index];
    const actualWidth =
      th?.getBoundingClientRect().width ?? state.columns[index].width;

    state.drag = {
      start: pointer.x.value,
      index,
      initialWidth: actualWidth,
    };
    // Sync state with actual DOM width, but for the last column,
    // exclude the extra space that was added to fill the container
    const isLastColumn = index === state.columns.length - 1;
    if (isLastColumn) {
      const containerW = options.containerRef?.value?.clientWidth || 0;
      const total = sumBy(state.columns, (col) => col.width);
      const extraWidth = Math.max(0, containerW - total);
      // Store actual width minus extra space
      state.columns[index].width = actualWidth - extraWidth;
    } else {
      state.columns[index].width = actualWidth;
    }
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
      scrollMaxX(options.containerRef.value);
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
    // For the first column (index column), use "1px" during auto-adjusting to shrink to content
    const autoWidth = index === 0 ? "1px" : "auto";

    // Calculate actual width including extra space for the last column
    let actualWidth = column.width;
    if (!state.isAutoAdjusting) {
      const containerW = options.containerRef?.value?.clientWidth || 0;
      const total = sumBy(state.columns, (col) => col.width);
      const extraWidth = Math.max(0, containerW - total);
      if (extraWidth > 0 && index === state.columns.length - 1) {
        actualWidth = column.width + extraWidth;
      }
    }

    const widthValue = state.isAutoAdjusting ? autoWidth : `${actualWidth}px`;
    return {
      style: {
        width: widthValue,
        minWidth: widthValue,
        maxWidth: widthValue,
        boxSizing: "border-box" as const,
      },
      class: state.isAutoAdjusting ? "" : "truncate",
      "data-index": index,
    };
  };

  const getColumnWidth = (index: number): number => {
    const column = state.columns[index];
    if (!column) return 0;

    // When effectiveWidth > totalWidth, add extra space to the last column
    const containerW = options.containerRef?.value?.clientWidth || 0;
    const total = sumBy(state.columns, (col) => col.width);
    const extraWidth = Math.max(0, containerW - total);

    // Add extra width to the last column
    if (extraWidth > 0 && index === state.columns.length - 1) {
      return column.width + extraWidth;
    }

    return column.width;
  };

  const totalWidth = computed(() => {
    return sumBy(state.columns, (col) => col.width);
  });

  // Effective width is the max of totalWidth and container width
  const effectiveWidth = computed(() => {
    const containerW = options.containerRef?.value?.clientWidth || 0;
    return Math.max(totalWidth.value, containerW);
  });

  const getTableProps = () => {
    const widthValue = state.isAutoAdjusting
      ? "auto"
      : `${effectiveWidth.value}px`;
    const tableLayout: "auto" | "fixed" = state.isAutoAdjusting
      ? "auto"
      : "fixed";
    return {
      style: {
        width: widthValue,
        tableLayout,
      },
    };
  };

  onBeforeUnmount(() => {
    toggleDragStyle(table, false);
  });

  return {
    reset,
    getColumnProps,
    getColumnWidth,
    getTableProps,
    totalWidth,
    effectiveWidth,
    startResizing,
  };
};

export default useTableResize;

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

// Measure text width using canvas for accurate calculation
// let measureCanvas: HTMLCanvasElement | null = null;
const measureTextWidth = (text: string, font: string): number => {
  const measureCanvas = document.createElement("canvas");
  const context = measureCanvas.getContext("2d");
  if (!context) return 0;
  context.font = font;
  return context.measureText(text).width;
};
