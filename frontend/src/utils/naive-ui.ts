import { pausableWatch, useElementSize } from "@vueuse/core";
import type { DataTableInst, VirtualListInst } from "naive-ui";
import { computed, type MaybeRef, ref, unref } from "vue";
import { nextAnimationFrame } from "./dom";

type AutoHeightDataTableOptions = {
  maxHeight: number | null;
};
const defaultAutoHeightDataTableOptions = (): AutoHeightDataTableOptions => ({
  maxHeight: null,
});

export const useAutoHeightDataTable = (
  data?: MaybeRef<unknown[]>,
  options?: MaybeRef<Partial<AutoHeightDataTableOptions>>
) => {
  const opts = computed(() => ({
    ...defaultAutoHeightDataTableOptions(),
    ...unref(options),
  }));
  const GAP_HEIGHT = 2;
  const dataTableRef = ref<DataTableInst>();
  const containerElRef = ref<HTMLElement>();
  const tableHeaderElRef = computed(
    () =>
      containerElRef.value?.querySelector("thead.n-data-table-thead") as
        | HTMLElement
        | undefined
  );
  const { height: containerHeight } = useElementSize(containerElRef);
  const { height: tableHeaderHeight } = useElementSize(tableHeaderElRef);
  const tableBodyHeight = ref(0);

  const { pause: pauseWatch, resume: resumeWatch } = pausableWatch(
    [() => unref(data), containerHeight, tableHeaderHeight, opts],
    async () => {
      if (containerHeight.value === 0 || tableHeaderHeight.value === 0) return;
      if (opts.value.maxHeight === null) {
        // The table height is always limited by the container height
        // and need not to be adjusted automatically
        tableBodyHeight.value =
          containerHeight.value - tableHeaderHeight.value - GAP_HEIGHT;
        return;
      }

      pauseWatch();
      const experimentalMaxBodyHeight =
        opts.value.maxHeight - tableHeaderHeight.value - GAP_HEIGHT;
      tableBodyHeight.value = experimentalMaxBodyHeight;
      await nextAnimationFrame();
      resumeWatch();
    },
    { immediate: true }
  );

  // Use this to avoid unnecessary initial rendering
  const layoutReady = computed(() => tableHeaderHeight.value > 0);

  const virtualListRef = computed(() => {
    // biome-ignore lint/suspicious/noExplicitAny: accessing internal naive-ui refs
    return (dataTableRef.value as any)?.$refs?.mainTableInstRef?.bodyInstRef // eslint-disable-line @typescript-eslint/no-explicit-any
      ?.virtualListRef as VirtualListInst | undefined;
  });

  const scrollerRef = computed(() => {
    // biome-ignore lint/suspicious/noExplicitAny: accessing internal naive-ui refs
    const getter = (dataTableRef.value as any)?.$refs.mainTableInstRef.$refs // eslint-disable-line @typescript-eslint/no-explicit-any
      .bodyInstRef.virtualListContainer;
    if (typeof getter === "function") {
      return getter() as HTMLElement | undefined;
    }
    return undefined;
  });

  return {
    dataTableRef,
    containerElRef,
    tableHeaderElRef,
    tableBodyHeight,
    layoutReady,
    virtualListRef,
    scrollerRef,
  };
};
