<template>
  <slot name="table" :list="state.changeHistoryList" :loading="state.loading" />

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

<script lang="ts" setup>
import { useSessionStorage } from "@vueuse/core";
import { stringify } from "qs";
import type { PropType } from "vue";
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { databaseServiceClient } from "@/grpcweb";
import { useIsLoggedIn } from "@/store";
import type { ComposedDatabase, SearchChangeHistoriesParams } from "@/types";
import type { ChangeHistory } from "@/types/proto/v1/database_service";

type LocalState = {
  loading: boolean;
  changeHistoryList: ChangeHistory[];
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

const props = defineProps({
  // A unique key to identify the session state.
  sessionKey: {
    type: String,
    required: true,
  },
  database: {
    type: Object as PropType<ComposedDatabase>,
    required: true,
  },
  searchChangeHistories: {
    type: Object as PropType<SearchChangeHistoriesParams>,
    default: () => ({}),
  },
  pageSize: {
    type: Number,
    default: 20, // Default to 20.
  },
  hideLoadMore: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "list:update", list: ChangeHistory[]): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  changeHistoryList: [],
  paginationToken: "",
});

const sessionState = useSessionStorage<SessionState>(props.sessionKey, {
  page: 1,
  updatedTs: 0,
});

const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const condition = computed(() => {
  const searchChangeHistories = {
    ...props.searchChangeHistories,
    limit: limit.value,
  };
  return stringify(searchChangeHistories, {
    arrayFormat: "repeat",
  });
});

const buildFilter = (search: SearchChangeHistoriesParams): string => {
  const filter: string[] = [];
  if (search.types && search.types.length > 0) {
    filter.push(`type = "${search.types.join(" | ")}"`);
  }
  if (search.tables && search.tables.length > 0) {
    filter.push(
      `table = "${search.tables.map((table) => `tableExists('${props.database.databaseName}', '${table.schema}', '${table.table}')`).join(" || ")}"`
    );
  }
  return filter.join(" && ");
};

const fetchData = async (refresh = false) => {
  state.loading = true;

  const isFirstFetch = state.paginationToken === "";
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      limit.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      limit.value;

  try {
    const { nextPageToken, changeHistories } =
      await databaseServiceClient.listChangeHistories({
        parent: props.database.name,
        filter: buildFilter(props.searchChangeHistories),
        pageSize: expectedRowCount,
        pageToken: state.paginationToken,
      });
    if (refresh) {
      state.changeHistoryList = changeHistories;
    } else {
      state.changeHistoryList.push(...changeHistories);
    }

    if (!isFirstFetch && changeHistories.length === expectedRowCount) {
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
watch(condition, refresh);
watch(isLoggedIn, () => {
  // Reset session when logged out.
  if (!isLoggedIn.value) {
    resetSession();
  }
});

watch(
  () => state.changeHistoryList,
  (list) => emit("list:update", list)
);
</script>
