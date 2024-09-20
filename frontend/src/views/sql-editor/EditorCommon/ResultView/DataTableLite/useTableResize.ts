import { useEventListener, usePointer } from "@vueuse/core";
import { sumBy } from "lodash-es";
import type { Ref } from "vue";
import { computed, onBeforeUnmount, reactive, unref, watch } from "vue";

export type TableResizeOptions = {
  scrollerRef: Ref<HTMLElement | undefined>;
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
  drag?: DragState;
};

const useTableResize = (options: TableResizeOptions) => {
  const state = reactive<LocalState>({
    columns: [],
    drag: undefined,
  });
  const pointer = usePointer();
  const scroller = options.scrollerRef;

  const containerWidth = computed(() => {
    return scroller.value?.scrollWidth || 0;
  });
  const tableWidth = computed(() => {
    return sumBy(state.columns, (col) => col.width);
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
    state.drag = undefined;

    const columnCount = unref(options.columnCount);
    state.columns = new Array(columnCount);

    for (let i = 0; i < columnCount; i++) {
      state.columns[i] = {
        width: options.defaultWidth,
      };
    }
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
    return {
      style: {
        width: `${tableWidth.value}px`,
      },
    };
  };

  onBeforeUnmount(() => {
    toggleDragStyle(scroller, false);
  });

  return {
    reset,
    state,
    tableWidth,
    getCellProps,
    getTableProps,
    startResizing,
  };
};

export default useTableResize;

const scrollMaxX = (elem: HTMLElement | undefined) => {
  if (!elem) return;
  const max = elem.scrollWidth;
  elem.scrollTo(max, elem.scrollTop);
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
