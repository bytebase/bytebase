import { useEventListener, usePointer } from "@vueuse/core";
import { sumBy } from "lodash-es";
import type { Ref } from "vue";
import { computed, onBeforeUnmount, reactive, watch } from "vue";

export type TableResizeOptions = {
  tableRef: Ref<HTMLTableElement | undefined>;
  containerRef: Ref<HTMLElement | null | undefined>;
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
    state.isAutoAdjusting = false;

    // Use auto layout to calculate the width of each column.
    table.value.style.tableLayout = "auto";
    const scale = containerWidth.value / table.value.scrollWidth;
    const thList = Array.from(table.value.querySelectorAll("th"));
    const columnCount = thList.length;
    if (columnCount === 1) {
      state.columns = [
        {
          width: containerWidth.value - 1,
        },
      ];
    } else {
      state.columns = thList
        .map((th) =>
          Math.max(
            th.getBoundingClientRect().width,
            th.getBoundingClientRect().width * scale
          )
        )
        .map((width) => ({ width }));
    }
    // After calculating the width, use fixed layout to keep the width stable.
    table.value.style.tableLayout = "fixed";
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
