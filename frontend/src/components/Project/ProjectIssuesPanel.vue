<template>
  <div class="flex flex-col gap-y-2">
    <IssueSearch
      v-model:params="state.params"
      :components="['searchbox', 'time-range', 'presets', 'status']"
    />

    <div class="relative min-h-80">
      <PagedTable
        ref="issuePagedTable"
        session-key="bb.issue-table.project-issues"
        :fetch-list="fetchIssueList"
      >
        <template #table="{ list, loading }">
          <IssueTableV1
            :bordered="true"
            :loading="loading"
            :issue-list="list"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRoute, useRouter } from "vue-router";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  useCurrentUserV1,
  useIssueV1Store,
  useRefreshIssueList,
} from "@/store";
import type { ComposedIssue } from "@/types";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { SearchParams, SearchScope } from "@/utils";
import {
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  extractProjectResourceName,
  mergeSearchParams,
} from "@/utils";
import { IssueSearch } from "../IssueV1/components";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  project: Project;
}>();

const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const issueStore = useIssueV1Store();
const issuePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedIssue>>>();

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      value: extractProjectResourceName(props.project.name),
      readonly: true,
    },
  ];
});

const defaultSearchParams = (): SearchParams => {
  const myEmail = me.value.email;
  return {
    query: "",
    scopes: [
      ...readonlyScopes.value,
      { id: "status", value: IssueStatus[IssueStatus.OPEN] },
      {
        id: "approval",
        value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
      },
      { id: "current-approver", value: myEmail },
    ],
  };
};

// Track the initial URL query string to distinguish between user-provided URLs
// and programmatic navigation to default state
const initialQueryString = (route.query.q as string) || null;

// Initialize params from URL query if present, otherwise use defaults
const getInitialParams = (): SearchParams => {
  if (initialQueryString) {
    const urlParams = buildSearchParamsBySearchText(initialQueryString);
    const readonlyParams: SearchParams = {
      query: "",
      scopes: [...readonlyScopes.value],
    };
    return mergeSearchParams(readonlyParams, urlParams);
  }
  return defaultSearchParams();
};

const state = reactive<LocalState>({
  params: getInitialParams(),
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

// Skip the first watch trigger since PagedTable already fetches on mount
let isFirstFilterWatch = true;
watch(
  () => JSON.stringify(mergedIssueFilter.value),
  () => {
    if (isFirstFilterWatch) {
      isFirstFilterWatch = false;
      return;
    }
    issuePagedTable.value?.refresh();
  }
);
useRefreshIssueList(() => issuePagedTable.value?.refresh());

watch(
  () => props.project.name,
  () => {
    state.params = defaultSearchParams();
  }
);

// Helper to check if params match the default preset
const isDefaultPreset = (params: SearchParams): boolean => {
  const defaultParams = defaultSearchParams();
  const paramsQuery = buildSearchTextBySearchParams(params);
  const defaultQuery = buildSearchTextBySearchParams(defaultParams);
  return paramsQuery === defaultQuery;
};

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
      if (isDefaultPreset(params) && !initialQueryString) {
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
