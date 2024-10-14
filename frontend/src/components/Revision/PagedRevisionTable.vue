<template>
  <slot name="table" :list="state.revisionList" :loading="state.loading" />

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
import type { PropType } from "vue";
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { databaseServiceClient } from "@/grpcweb";
import { useIsLoggedIn } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { Revision } from "@/types/proto/v1/database_service";

type LocalState = {
  loading: boolean;
  revisionList: Revision[];
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
  (event: "list:update", list: Revision[]): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  revisionList: [],
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

const fetchData = async (refresh = false) => {
  state.loading = true;

  const isFirstFetch = state.paginationToken === "";
  const expectedRowCount = isFirstFetch
    ? // Load one or more page for the first fetch to restore the session
      limit.value * sessionState.value.page
    : // Always load one page if NOT the first fetch
      limit.value;

  try {
    const { nextPageToken, revisions } =
      await databaseServiceClient.listRevisions({
        parent: props.database.name,
        pageSize: expectedRowCount,
        pageToken: state.paginationToken,
      });
    if (refresh) {
      state.revisionList = revisions;
    } else {
      state.revisionList.push(...revisions);
    }

    if (!isFirstFetch && revisions.length === expectedRowCount) {
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
  () => state.revisionList,
  (list) => emit("list:update", list)
);
</script>
