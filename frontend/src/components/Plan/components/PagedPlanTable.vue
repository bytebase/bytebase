<template>
  <slot name="table" :plan-list="state.planList" :loading="state.loading" />

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
import { isEqual } from "lodash-es";
import type { PropType } from "vue";
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useIsLoggedIn, useRefreshPlanList } from "@/store";
import {
  usePlanStore,
  type ListPlanParams,
  type PlanFind,
} from "@/store/modules/v1/plan";
import type { ComposedPlan } from "@/types/v1/issue/plan";

type LocalState = {
  loading: boolean;
  planList: ComposedPlan[];
  paginationToken: string;
  loadingMore: boolean;
};

/**
 * It's complex and dangerous to cache the plan list.
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
  planFind: {
    type: Object as PropType<PlanFind>,
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
  loading: {
    type: Boolean,
    default: false,
  },
  loadingMore: {
    type: Boolean,
    default: false,
  },
});
const emit = defineEmits<{
  (event: "update:loading", loading: boolean): void;
  (event: "update:loading-more", loadingMore: boolean): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  planList: [],
  paginationToken: "",
  loadingMore: false,
});

const sessionState = useSessionStorage<SessionState>(
  `bb.page-plan-table.${props.sessionKey}`,
  {
    page: 1,
    updatedTs: 0,
  }
);

const planStore = usePlanStore();
const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const latestParams = computed((): ListPlanParams => {
  return {
    find: props.planFind,
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

  const params: ListPlanParams = {
    find: props.planFind,
    pageSize: props.pageSize,
    pageToken: state.paginationToken,
  };
  const request = planStore.searchPlans(params);

  request
    .then(({ nextPageToken, plans }) => {
      if (!isEqual(params, latestParams.value)) {
        // The search params changed during fetching data
        // The result is outdated
        // Give up
        return;
      }
      if (refresh) {
        state.planList = plans;
      } else {
        state.planList.push(...plans);
      }

      if (plans.length >= expectedRowCount && !isFirstFetch) {
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
watch(() => JSON.stringify(props.planFind), refresh);
watch(isLoggedIn, () => {
  // Reset session when logged out.
  if (!isLoggedIn.value) {
    resetSession();
  }
});

useRefreshPlanList(() => {
  refresh();
});

watch(
  () => state.loading,
  (loading) => {
    emit("update:loading", loading);
  },
  { immediate: true }
);
watch(
  () => state.loadingMore,
  (loadingMore) => {
    emit("update:loading-more", loadingMore);
  },
  { immediate: true }
);
</script>
