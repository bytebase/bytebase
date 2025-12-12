<template>
  <div :key="viewId" class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="['searchbox', 'time-range', 'presets', 'status']"
      :default-params="computedDefaultParams"
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
  `bb.components.${WORKSPACE_ROUTE_MY_ISSUES}.id`,
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

// Initialize with empty params - AdvancedSearch will set from cache or defaults
const state = reactive<LocalState>({
  params: { query: "", scopes: [] },
});

// Always provide defaultParams for AdvancedSearch to use as fallback
const computedDefaultParams = computed(() => defaultSearchParams());

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

// Track the initial URL query string to distinguish between user-provided URLs
// and programmatic navigation to default state
const initialQueryString = ref<string | null>(null);

// Initialize params from URL query on mount
onMounted(() => {
  const queryString = route.query.q as string;
  initialQueryString.value = queryString || null;

  if (queryString) {
    const urlParams = buildSearchParamsBySearchText(queryString);
    // Use URL params directly, don't merge with defaults
    state.params = urlParams;
  }
  // No else - keep URL clean for default preset
});

// Sync params to URL query when params change
let isUpdatingFromUrl = false;
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
      // Special case: if at default preset and we didn't start with a URL, keep URL clean
      if (isDefaultPreset(params) && !initialQueryString.value) {
        // Remove URL query
        if (route.query.q) {
          isUpdatingFromUrl = true;
          router
            .replace({
              query: {
                ...route.query,
                q: undefined,
              },
            })
            .finally(() => {
              isUpdatingFromUrl = false;
            });
        }
      } else {
        // Update URL normally
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
    }
  },
  { deep: true }
);
</script>
