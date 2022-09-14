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
import { buildQueryListByIssueFind, useIssueStore } from "@/store";
import { Issue, IssueFind } from "@/types";
import { computed, PropType, reactive, watch } from "vue";

type LocalState = {
  loading: boolean;
  issueList: Issue[];
  paginationToken: string;
  hasMore: boolean;
};

const MAX_PAGE_SIZE = 1000;

const props = defineProps({
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

const issueStore = useIssueStore();

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
  state.loading = true;
  issueStore
    .fetchPagedIssueList({
      ...props.issueFind,
      limit: limit.value,
      token: state.paginationToken,
    })
    .then(({ nextToken, issueList }) => {
      if (refresh) {
        state.issueList = issueList;
      } else {
        state.issueList.push(...issueList);
      }

      if (issueList.length === 0) {
        state.hasMore = false;
      }

      state.paginationToken = nextToken;
    })
    .finally(() => {
      state.loading = false;
    });
};

const refresh = () => {
  state.paginationToken = "";
  state.hasMore = true;
  fetchData(true);
};

const fetchNextPage = () => {
  fetchData(false);
};

fetchData(true);
watch(condition, refresh);
</script>
