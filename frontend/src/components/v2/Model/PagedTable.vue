<template>
  <div class="flex flex-col gap-y-4">
    <slot
      name="table"
      :list="dataList"
      :loading="isLoading"
      :sorters="sorters"
      :on-sorters-update="onSortersUpdate"
    />

    <div :class="['flex items-center justify-end gap-x-2', footerClass]">
      <div class="flex items-center gap-x-2">
        <div class="textinfolabel">
          {{ $t("common.rows-per-page") }}
        </div>
        <NSelect
          :value="pageSize"
          style="width: 5rem"
          size="small"
          :options="pageSizeOptions"
          @update:value="onPageSizeChange"
        />
      </div>

      <NButton
        v-if="!hideLoadMore && hasMore"
        quaternary
        size="small"
        :loading="isFetching"
        @click="loadMore"
      >
        <span class="textinfolabel">
          {{ $t("common.load-more") }}
        </span>
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup generic="T extends { name: string }">
import { useDebounceFn } from "@vueuse/core";
import { sortBy, uniq } from "lodash-es";
import { type DataTableSortState, NButton, NSelect } from "naive-ui";
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import { useAuthStore, useCurrentUserV1 } from "@/store";
import { getDefaultPagination, useDynamicLocalStorage } from "@/utils";

// ============================================================================
// Props & Emits
// ============================================================================

const props = withDefaults(
  defineProps<{
    sessionKey: string;
    hideLoadMore?: boolean;
    footerClass?: string;
    debounce?: number;
    fetchList: (params: {
      pageSize: number;
      pageToken: string;
      refresh?: boolean;
      orderBy?: string;
    }) => Promise<{ nextPageToken?: string; list: T[] }>;
    orderKeys?: string[];
  }>(),
  {
    hideLoadMore: false,
    footerClass: "",
    debounce: 500,
    orderKeys: () => [],
  }
);

const emit = defineEmits<{
  "list:update": [list: T[]];
}>();

// ============================================================================
// Stores
// ============================================================================

const authStore = useAuthStore();
const currentUser = useCurrentUserV1();

// ============================================================================
// State
// ============================================================================

// Core data state
const dataList = shallowRef<T[]>([]);
const nextPageToken = ref("");

// Fetch state
const isFetching = ref(false);
const ready = ref(false);
const error = ref<Error | null>(null);

// Request management
let abortController: AbortController | null = null;

// ============================================================================
// Computed
// ============================================================================

// Loading state: true if fetching OR if no fetch has completed yet
// This prevents showing "No Data" before the first request finishes
const isLoading = computed(() => isFetching.value || !ready.value);

// Whether there are more pages to load
const hasMore = computed(() => Boolean(nextPageToken.value));

// ============================================================================
// Sorting
// ============================================================================

const sorters = ref<DataTableSortState[]>(
  props.orderKeys.map((key) => ({
    columnKey: key,
    order: false,
    sorter: true,
  }))
);

const orderBy = computed(() =>
  sorters.value
    .filter((sorter) => sorter.order)
    .map((sorter) => {
      const key = sorter.columnKey.toString();
      const order = sorter.order === "ascend" ? "asc" : "desc";
      return `${key} ${order}`;
    })
    .join(", ")
);

const onSortersUpdate = (
  sortStates: DataTableSortState[] | DataTableSortState | null
) => {
  if (!sortStates) return;

  const states = Array.isArray(sortStates) ? sortStates : [sortStates];
  for (const state of states) {
    const index = sorters.value.findIndex(
      (s) => s.columnKey === state.columnKey
    );
    if (index >= 0) {
      sorters.value[index] = state;
    }
  }
};

// ============================================================================
// Pagination
// ============================================================================

const pageSizeOptions = computed(() => {
  const defaultPageSize = getDefaultPagination();
  return sortBy(uniq([defaultPageSize, 50, 100, 200, 500])).map((num) => ({
    value: num,
    label: String(num),
  }));
});

const sessionState = useDynamicLocalStorage<{ pageSize: number }>(
  computed(() => `${props.sessionKey}.${currentUser.value.name}`),
  { pageSize: pageSizeOptions.value[0].value }
);

const pageSize = computed(() => {
  const stored = sessionState.value.pageSize;
  const defaultSize = pageSizeOptions.value[0].value;

  // Guard against invalid values from corrupted localStorage
  if (
    !Number.isFinite(stored) ||
    !pageSizeOptions.value.some((o) => o.value === stored)
  ) {
    return defaultSize;
  }
  return Math.max(defaultSize, stored);
});

// ============================================================================
// Data Fetching
// ============================================================================

const fetchData = async (isRefresh: boolean) => {
  if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
    return;
  }

  // Cancel any in-flight request
  abortController?.abort();
  abortController = new AbortController();

  isFetching.value = true;
  error.value = null;

  try {
    const result = await props.fetchList({
      pageSize: pageSize.value,
      pageToken: isRefresh ? "" : nextPageToken.value,
      refresh: isRefresh,
      orderBy: orderBy.value,
    });

    // Check if request was aborted
    if (abortController.signal.aborted) {
      return;
    }

    if (isRefresh) {
      dataList.value = result.list;
    } else {
      dataList.value = [...dataList.value, ...result.list];
    }

    nextPageToken.value = result.nextPageToken ?? "";
    ready.value = true;
  } catch (e) {
    // Ignore abort errors
    if (e instanceof Error && e.name === "AbortError") {
      return;
    }
    console.error(e);
    error.value = e instanceof Error ? e : new Error(String(e));
    ready.value = true;
  } finally {
    isFetching.value = false;
  }
};

// Debounced refresh for internal triggers (watchers)
const debouncedRefresh = useDebounceFn(() => {
  nextPageToken.value = "";
  fetchData(true);
}, props.debounce);

// Immediate refresh that returns a Promise - for external callers
const refresh = async () => {
  nextPageToken.value = "";
  await fetchData(true);
};

const loadMore = async () => {
  if (!hasMore.value || isFetching.value) return;
  await fetchData(false);
};

// ============================================================================
// Event Handlers
// ============================================================================

const onPageSizeChange = (size: number) => {
  sessionState.value.pageSize = size;
  debouncedRefresh();
};

// ============================================================================
// Cache Management
// ============================================================================

const updateCache = (items: T[]) => {
  const updated = [...dataList.value];
  let hasChanges = false;

  for (const item of items) {
    const index = updated.findIndex((d) => d.name === item.name);
    if (index >= 0) {
      updated[index] = item;
      hasChanges = true;
    }
  }

  if (hasChanges) {
    dataList.value = updated;
  }
};

const removeCache = (item: T) => {
  const index = dataList.value.findIndex((d) => d.name === item.name);
  if (index >= 0) {
    const updated = [...dataList.value];
    updated.splice(index, 1);
    dataList.value = updated;
  }
};

// ============================================================================
// Lifecycle
// ============================================================================

onMounted(() => {
  fetchData(true);
});

onUnmounted(() => {
  abortController?.abort();
});

// ============================================================================
// Watchers
// ============================================================================

// Re-fetch on auth changes
watch(
  () => authStore.authSessionKey,
  () => {
    if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
      return;
    }
    sessionState.value = { pageSize: pageSize.value };
    debouncedRefresh();
  }
);

// Emit list updates
watch(dataList, (list) => emit("list:update", list));

// Re-fetch when sort order changes (debounced to handle rapid changes)
watch(orderBy, () => debouncedRefresh());

// ============================================================================
// Exposed API
// ============================================================================

defineExpose({
  // Methods
  refresh,
  loadMore,
  updateCache,
  removeCache,

  // State (readonly access)
  dataList,
  orderBy,
  isLoading,
  isFetching,
  hasMore,
  ready,
  error,
});
</script>
