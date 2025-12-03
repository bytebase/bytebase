<template>
  <div :key="viewId" class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :default-params="defaultSearchParams()"
      :components="['searchbox', 'time-range', 'presets', 'status']"
      class="px-4 pb-2"
    />

    <div class="relative min-h-80">
      <PagedTable
        ref="issuePagedTable"
        session-key="bb.issue-table.my-issues"
        :fetch-list="fetchIssueList"
      >
        <template #table="{ list, loading }">
          <IssueTableV1
            class="border-x-0"
            :loading="loading"
            :issue-list="list"
            :highlight-text="state.params.query"
            :show-project="true"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { computed, onMounted, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRoute, useRouter } from "vue-router";
import { IssueSearch } from "@/components/IssueV1/components";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";
import {
  useCurrentUserV1,
  useIssueV1Store,
  useRefreshIssueList,
} from "@/store";
import { type ComposedIssue } from "@/types";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams } from "@/utils";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
} from "@/utils";
import { getComponentIdLocalStorageKey } from "@/utils/localStorage";

interface LocalState {
  params: SearchParams;
}

const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const issueStore = useIssueV1Store();
const issuePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedIssue>>>();

const viewId = useLocalStorage<string>(
  getComponentIdLocalStorageKey(WORKSPACE_ROUTE_MY_ISSUES),
  ""
);

const defaultSearchParams = (): SearchParams => {
  const myEmail = me.value.email;
  return {
    query: "",
    scopes: [
      { id: "status", value: IssueStatus[IssueStatus.OPEN] },
      {
        id: "approval",
        value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
      },
      { id: "current-approver", value: myEmail },
    ],
  };
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(state.params);
});

const fetchIssueList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, issues } = await issueStore.listIssues({
    find: mergedIssueFilter.value,
    pageSize,
    pageToken,
  });
  return {
    nextPageToken,
    list: issues,
  };
};

watch(
  () => JSON.stringify(mergedIssueFilter.value),
  () => issuePagedTable.value?.refresh()
);
useRefreshIssueList(() => issuePagedTable.value?.refresh());

// Helper to check if params match the default preset
const isDefaultPreset = (params: SearchParams): boolean => {
  const defaultParams = defaultSearchParams();
  const paramsQuery = buildSearchTextBySearchParams(params);
  const defaultQuery = buildSearchTextBySearchParams(defaultParams);
  return paramsQuery === defaultQuery;
};

// Initialize params from URL query on mount
onMounted(() => {
  const queryString = route.query.q as string;
  if (queryString) {
    const urlParams = buildSearchParamsBySearchText(queryString);
    state.params = urlParams;
  }
});

// Sync URL to params when route query changes
let isUpdatingFromUrl = false;
watch(
  () => route.query.q as string | undefined,
  (newQuery) => {
    if (isUpdatingFromUrl) {
      return;
    }

    // When URL query is set, update params from URL
    // (When URL query is cleared, AdvancedSearch handles cache vs defaults)
    if (newQuery) {
      state.params = buildSearchParamsBySearchText(newQuery);
    }
  }
);

// Sync params to URL query when params change
watch(
  () => state.params,
  (params) => {
    if (isUpdatingFromUrl) {
      return;
    }

    const queryString = buildSearchTextBySearchParams(params);
    const currentQuery = route.query.q as string;

    // Only update URL if query string has actually changed
    if (queryString !== currentQuery) {
      // Special case: if at default preset and URL is clean, keep URL clean
      if (isDefaultPreset(params) && !currentQuery) {
        return;
      }
      // Update URL
      isUpdatingFromUrl = true;
      router
        .replace({
          query: {
            ...route.query,
            q: queryString || undefined,
          },
        })
        .finally(() => {
          isUpdatingFromUrl = false;
        });
    }
  },
  { deep: true }
);
</script>
