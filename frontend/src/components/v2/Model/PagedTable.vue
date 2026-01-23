<template>
  <div class="flex flex-col gap-y-4">
    <slot name="table" :list="dataList" :loading="state.loading" />

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
import { NButton, NSelect } from "naive-ui";
import { computed, type Ref, reactive, ref, watch } from "vue";
import { useAuthStore, useCurrentUserV1 } from "@/store";
import { getDefaultPagination, useDynamicLocalStorage } from "@/utils";

type LocalState = {
  loading: boolean;
  paginationToken: string;
};

type SessionState = {
  // How many times the user clicks the "load more" button.
  page: number;
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
    }) => Promise<{ nextPageToken?: string; list: T[] }>;
  }>(),
  {
    hideLoadMore: false,
    footerClass: "",
    debounce: 500,
  }
);

const emit = defineEmits<{
  (event: "list:update", list: T[]): void;
}>();

const authStore = useAuthStore();
const currentUser = useCurrentUserV1();

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
    page: 1,
    updatedTs: 0,
    pageSize: options.value[0].value,
  }
);

const pageSize = computed(() => {
  const sizeInSession = sessionState.value.pageSize ?? 0;
  // Guard against NaN/invalid values from corrupted localStorage data
  if (
    !Number.isFinite(sizeInSession) ||
    !options.value.find((o) => o.value === sizeInSession)
  ) {
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

  const isFirstFetch = state.paginationToken === "";
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      pageSize.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      pageSize.value;

  try {
    const { nextPageToken, list } = await props.fetchList({
      pageSize: expectedRowCount,
      pageToken: state.paginationToken,
      refresh,
    });
    if (refresh) {
      dataList.value = list;
    } else {
      dataList.value.push(...list);
    }

    if (!isFirstFetch && list.length === expectedRowCount) {
      // If we didn't reach the end, memorize we've clicked the "load more" button.
      sessionState.value.page++;
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
    page: 1,
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
});
</script>
