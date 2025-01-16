<template>
  <slot name="table" :list="dataList" :loading="state.loading" />

  <div
    v-if="state.loading"
    class="flex items-center justify-center py-2 text-gray-400 text-sm"
  >
    <BBSpin />
  </div>

  <slot
    v-if="!state.loading"
    name="more"
    :has-more="!!state.paginationToken"
    :fetch-next-page="fetchNextPage"
  >
    <div
      v-if="!hideLoadMore && pageSize > 0 && state.paginationToken"
      class="flex items-center justify-center py-2 text-gray-400 text-sm hover:bg-gray-200 cursor-pointer"
      @click="fetchNextPage"
    >
      {{ $t("common.load-more") }}
    </div>
  </slot>
</template>

<script lang="ts" setup generic="T extends object">
import { useSessionStorage } from "@vueuse/core";
import { computed, reactive, watch, ref, type Ref } from "vue";
import { BBSpin } from "@/bbkit";
import { useIsLoggedIn } from "@/store";

type LocalState = {
  loading: boolean;
  paginationToken: string;
};

type SessionState = {
  // How many times the user clicks the "load more" button.
  page: number;
  // Help us to check if the session is outdated.
  updatedTs: number;
};

const MAX_PAGE_SIZE = 1000;
const SESSION_LIFE = 1 * 60 * 1000; // 1 minute

const props = withDefaults(
  defineProps<{
    // A unique key to identify the session state.
    sessionKey: string;
    pageSize?: number;
    hideLoadMore?: boolean;
    fetchList: (params: {
      pageSize: number;
      pageToken: string;
    }) => Promise<{ nextPageToken: string; list: T[] }>;
  }>(),
  {
    pageSize: 10,
    hideLoadMore: false,
  }
);

const emit = defineEmits<{
  (event: "list:update", list: T[]): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  paginationToken: "",
});

// https://stackoverflow.com/questions/69813587/vue-unwraprefsimplet-generics-type-cant-assignable-to-t-at-reactive
const dataList = ref([]) as Ref<T[]>;

const sessionState = useSessionStorage<SessionState>(props.sessionKey, {
  page: 1,
  updatedTs: 0,
});

const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const fetchData = async (refresh = false) => {
  if (!isLoggedIn.value) {
    return;
  }

  state.loading = true;

  const isFirstFetch = state.paginationToken === "";
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      limit.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      limit.value;

  try {
    const { nextPageToken, list } = await props.fetchList({
      pageSize: expectedRowCount,
      pageToken: state.paginationToken,
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
    state.paginationToken = nextPageToken;
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
  };
};

const refresh = () => {
  state.paginationToken = "";
  resetSession();
  fetchData(true);
};

const fetchNextPage = () => {
  fetchData(false);
};

if (Date.now() - sessionState.value.updatedTs > SESSION_LIFE) {
  // Reset session if it's outdated.
  resetSession();
}
fetchData(true);
watch(isLoggedIn, () => {
  // Reset session when logged out.
  if (!isLoggedIn.value) {
    resetSession();
  }
});

watch(
  () => dataList.value,
  (list) => emit("list:update", list)
);

defineExpose({
  refresh,
});
</script>
