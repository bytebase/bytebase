<template>
  <slot name="table" :list="state.auditLogList" :loading="state.loading" />

  <div
    v-if="state.loading"
    class="flex items-center justify-center py-2 text-gray-400 text-sm"
  >
    <BBSpin />
  </div>

  <slot
    v-if="!state.loading"
    name="more"
    :has-more="state.hasMore"
    :fetch-next-page="fetchNextPage"
  >
    <div
      v-if="!hideLoadMore && pageSize > 0 && state.hasMore"
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
import { computed, PropType, reactive, watch } from "vue";
import { useIsLoggedIn, useActivityV1Store } from "@/store";
import { FindActivityMessage } from "@/types";
import { LogEntity } from "@/types/proto/v1/logging_service";

type LocalState = {
  loading: boolean;
  auditLogList: LogEntity[];
  paginationToken: string;
  hasMore: boolean;
};

/**
 * It's complex and dangerous to cache the activity list.
 * So we just memorize how many times the user clicks the "load more" button.
 * And load the first N pages in the first fetch.
 * E.g., the user clicked "load more" 4 times, then the first time will set limit
 *   to `pageSize * 5`.
 */
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
  activityFind: {
    type: Object as PropType<FindActivityMessage>,
    default: () => ({
      order: "desc",
    }),
  },
  pageSize: {
    type: Number,
    default: 10,
  },
  hideLoadMore: {
    type: Boolean,
    default: false,
  },
});

const state = reactive<LocalState>({
  loading: false,
  auditLogList: [],
  paginationToken: "",
  hasMore: true,
});

const sessionState = useSessionStorage<SessionState>(props.sessionKey, {
  page: 1,
  updatedTs: 0,
});

const activityV1Store = useActivityV1Store();
const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const condition = computed(() => {
  const activityFind = {
    ...props.activityFind,
    limit: limit.value,
  };
  return stringify(activityFind, {
    arrayFormat: "repeat",
  });
});

const fetchData = async (refresh = false) => {
  state.loading = true;

  const isFirstFetch = state.paginationToken === "";
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      limit.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      limit.value;

  try {
    const { nextPageToken, logEntities } =
      await activityV1Store.fetchActivityList({
        ...props.activityFind,
        pageSize: expectedRowCount,
        pageToken: state.paginationToken,
      });
    if (refresh) {
      state.auditLogList = logEntities;
    } else {
      state.auditLogList.push(...logEntities);
    }

    if (logEntities.length < expectedRowCount) {
      state.hasMore = false;
    } else if (!isFirstFetch) {
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
  state.hasMore = true;
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
</script>
