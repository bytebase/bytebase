<template>
  <slot name="table" :issue-list="state.issueList" :loading="state.loading" />

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
import { computed, PropType, reactive, watch } from "vue";
import {
  buildQueryListByIssueFind,
  useIsLoggedIn,
  useIssueStore,
  useRefreshIssueList,
} from "@/store";
import { Issue, IssueFind } from "@/types";

type LocalState = {
  loading: boolean;
  issueList: Issue[];
  paginationToken: string;
  hasMore: boolean;
};

/**
 * It's complex and dangerous to cache the issue list.
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
  issueFind: {
    type: Object as PropType<IssueFind>,
    default: undefined,
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
  issueList: [],
  paginationToken: "",
  hasMore: true,
});

const sessionState = useSessionStorage<SessionState>(
  `bb.page-issue-table.${props.sessionKey}`,
  {
    page: 1,
    updatedTs: 0,
  }
);

const issueStore = useIssueStore();
const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const condition = computed(() => {
  const issueFind = {
    ...props.issueFind,
    limit: limit.value,
  };
  return buildQueryListByIssueFind(issueFind).join("&");
});

const fetchData = (refresh = false) => {
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

  issueStore
    .fetchPagedIssueList({
      ...props.issueFind,
      limit: expectedRowCount,
      token: state.paginationToken,
    })
    .then(({ nextToken, issueList }) => {
      if (refresh) {
        state.issueList = issueList;
      } else {
        state.issueList.push(...issueList);
      }

      if (issueList.length < expectedRowCount) {
        state.hasMore = false;
      } else if (!isFirstFetch) {
        // If we didn't reach the end, memorize we've clicked the "load more" button.
        sessionState.value.page++;
      }

      sessionState.value.updatedTs = Date.now();
      state.paginationToken = nextToken;
    })
    .finally(() => {
      state.loading = false;
    });
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

useRefreshIssueList(() => {
  refresh();
});
</script>
