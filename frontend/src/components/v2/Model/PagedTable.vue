<template>
  <div class="flex flex-col gap-y-4">
    <slot
      name="table"
      :list="dataList"
      :loading="state.loading"
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
          :size="'small'"
          :options="options"
          @update:value="onPageSizeChange"
        />
      </div>

      <NButton
        v-if="!hideLoadMore && state.paginationToken"
        quaternary
        :size="'small'"
        :loading="state.loading"
        @click="fetchNextPage"
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
import { computed, type Ref, reactive, ref, watch } from "vue";
import { useAuthStore, useCurrentUserV1 } from "@/store";
import { getDefaultPagination, useDynamicLocalStorage } from "@/utils";

type LocalState = {
  loading: boolean;
  paginationToken: string;
};

type SessionState = {
  // Help us to check if the session is outdated.
  updatedTs: number;
  pageSize: number;
};

const props = withDefaults(
  defineProps<{
    // A unique key to identify the session state.
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
  (event: "list:update", list: T[]): void;
}>();

const authStore = useAuthStore();
const currentUser = useCurrentUserV1();
const sorters = ref<DataTableSortState[]>(
  props.orderKeys.map((key) => ({
    columnKey: key,
    order: false,
    sorter: true,
  }))
);

const onSortersUpdate = (
  sortStates: DataTableSortState[] | DataTableSortState | null
) => {
  if (!sortStates) {
    return;
  }
  let states: DataTableSortState[] = [];
  if (Array.isArray(sortStates)) {
    states = sortStates;
  } else {
    states = [sortStates];
  }
  for (const sortState of states) {
    const sorterIndex = sorters.value.findIndex(
      (s) => s.columnKey === sortState.columnKey
    );
    if (sorterIndex >= 0) {
      sorters.value[sorterIndex] = sortState;
    }
  }
};

const orderBy = computed(() => {
  return sorters.value
    .filter((sorter) => sorter.order)
    .map((sorter) => {
      const key = sorter.columnKey.toString();
      const order = sorter.order == "ascend" ? "asc" : "desc";
      return `${key} ${order}`;
    })
    .join(", ");
});

const options = computed(() => {
  const defaultPageSize = getDefaultPagination();
  const list = [defaultPageSize, 50, 100, 200, 500];
  return sortBy(uniq(list)).map((num) => ({
    value: num,
    label: `${num}`,
  }));
});

const state = reactive<LocalState>({
  loading: false,
  paginationToken: "",
});

// https://stackoverflow.com/questions/69813587/vue-unwraprefsimplet-generics-type-cant-assignable-to-t-at-reactive
const dataList = ref([]) as Ref<T[]>;

const sessionState = useDynamicLocalStorage<SessionState>(
  computed(() => `${props.sessionKey}.${currentUser.value.name}`),
  {
    updatedTs: 0,
    pageSize: options.value[0].value,
  }
);

const pageSize = computed(() => {
  const sizeInSession = sessionState.value.pageSize ?? 0;
  if (!options.value.find((o) => o.value === sizeInSession)) {
    return options.value[0].value;
  }
  return Math.max(options.value[0].value, sizeInSession);
});

const onPageSizeChange = (size: number) => {
  sessionState.value.pageSize = size;
  refresh();
};

const fetchData = async (refresh = false) => {
  if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
    return;
  }

  state.loading = true;

  try {
    const { nextPageToken, list } = await props.fetchList({
      pageSize: pageSize.value,
      pageToken: state.paginationToken,
      refresh,
      orderBy: orderBy.value,
    });
    if (refresh) {
      dataList.value = list;
    } else {
      dataList.value.push(...list);
    }

    sessionState.value.updatedTs = Date.now();
    state.paginationToken = nextPageToken ?? "";
  } catch (e) {
    console.error(e);
  } finally {
    state.loading = false;
  }
};

const resetSession = () => {
  sessionState.value = {
    updatedTs: 0,
    pageSize: pageSize.value,
  };
};

const refresh = async () => {
  state.paginationToken = "";
  await fetchData(true);
};

const fetchNextPage = () => {
  fetchData(false);
};

fetchData(true);

watch(
  () => authStore.authSessionKey,
  () => {
    if (!authStore.isLoggedIn || authStore.unauthenticatedOccurred) {
      return;
    }
    // Reset session when logging status changed.
    resetSession();
    refresh();
  }
);

watch(
  () => dataList.value,
  (list) => emit("list:update", list)
);

watch(
  () => orderBy.value,
  () => refresh()
);

const updateCache = (data: T[]) => {
  for (const item of data) {
    const index = dataList.value.findIndex((d) => d.name === item.name);
    if (index >= 0) {
      dataList.value[index] = item;
    }
  }
};

const removeCache = (data: T) => {
  const index = dataList.value.findIndex((d) => d.name === data.name);
  dataList.value.splice(index, 1);
};

defineExpose({
  refresh: useDebounceFn(async () => {
    await refresh();
  }, props.debounce),
  updateCache,
  removeCache,
  dataList,
  orderBy,
});
</script>
