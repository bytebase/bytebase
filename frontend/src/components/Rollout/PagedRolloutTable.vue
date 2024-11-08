<template>
  <slot
    name="table"
    :rollout-list="state.rolloutList"
    :loading="state.loading"
  />

  <div
    v-if="state.loadingMore"
    class="flex items-center justify-center py-2 text-gray-400 text-sm"
  >
    <BBSpin />
  </div>

  <slot
    v-if="!state.loadingMore"
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
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useIsLoggedIn, useRolloutStore } from "@/store";
import type { ComposedRollout, Pagination } from "@/types";

type LocalState = {
  loading: boolean;
  rolloutList: ComposedRollout[];
  paginationToken: string;
  loadingMore: boolean;
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
  project: {
    type: String,
    required: true,
  },
  pageSize: {
    type: Number,
    default: 50,
  },
  hideLoadMore: {
    type: Boolean,
    default: false,
  },
});

const state = reactive<LocalState>({
  loading: false,
  rolloutList: [],
  paginationToken: "",
  loadingMore: false,
});

const sessionState = useSessionStorage<SessionState>(
  `bb.paged-rollout-data-table.${props.sessionKey}`,
  {
    page: 1,
    updatedTs: 0,
  }
);

const rolloutStore = useRolloutStore();
const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const latestPagination = computed((): Pagination => {
  return {
    pageSize: props.pageSize,
    pageToken: state.paginationToken,
  };
});

const fetchData = (refresh = false) => {
  if (!isLoggedIn.value) {
    return;
  }

  state.loading = true;

  const isFirstFetch = state.paginationToken === "";
  if (!isFirstFetch) {
    state.loadingMore = true;
  }
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      limit.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      limit.value;

  const request = rolloutStore.fetchRolloutsByProject(
    props.project,
    latestPagination.value
  );

  request
    .then(({ nextPageToken, rollouts }) => {
      if (refresh) {
        state.rolloutList = rollouts;
      } else {
        state.rolloutList.push(...rollouts);
      }

      if (rollouts.length >= expectedRowCount && !isFirstFetch) {
        // If we didn't reach the end, memorize we've clicked the "load more" button.
        sessionState.value.page++;
      }

      sessionState.value.updatedTs = Date.now();
      state.paginationToken = nextPageToken;
    })
    .finally(() => {
      state.loading = false;
      state.loadingMore = false;
    });
};

const resetSession = () => {
  sessionState.value = {
    page: 1,
    updatedTs: 0,
  };
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
</script>
