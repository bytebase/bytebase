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
import { useIsLoggedIn, useAuditLogStore } from "@/store";
import type { SearchAuditLogsParams } from "@/types";
import type { AuditLog } from "@/types/proto/v1/audit_log_service";

type LocalState = {
  loading: boolean;
  auditLogList: AuditLog[];
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
  searchAuditLogs: {
    type: Object as PropType<SearchAuditLogsParams>,
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

const emit = defineEmits<{
  (event: "list:update", list: AuditLog[]): void;
}>();

const state = reactive<LocalState>({
  loading: false,
  auditLogList: [],
  paginationToken: "",
});

const sessionState = useSessionStorage<SessionState>(props.sessionKey, {
  page: 1,
  updatedTs: 0,
});

const auditLogStore = useAuditLogStore();
const isLoggedIn = useIsLoggedIn();

const limit = computed(() => {
  if (props.pageSize <= 0) return MAX_PAGE_SIZE;
  return props.pageSize;
});

const condition = computed(() => {
  const searchAuditLogs = {
    ...props.searchAuditLogs,
    limit: limit.value,
  };
  return stringify(searchAuditLogs, {
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
    const { nextPageToken, auditLogs } = await auditLogStore.fetchAuditLogs({
      ...props.searchAuditLogs,
      pageSize: expectedRowCount,
      pageToken: state.paginationToken,
    });
    if (refresh) {
      state.auditLogList = auditLogs;
    } else {
      state.auditLogList.push(...auditLogs);
    }

    if (!isFirstFetch && auditLogs.length === expectedRowCount) {
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
  () => state.auditLogList,
  (list) => emit("list:update", list)
);
</script>
