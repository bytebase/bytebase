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
            :issue-list="applyUIIssueFilter(list, mergedUIIssueFilter)"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { type UseStorageOptions } from "@vueuse/core";
import { computed, onMounted, reactive, ref, watch } from "vue";
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
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { SearchParams, SearchScope, SemanticIssueStatus } from "@/utils";
import {
  applyUIIssueFilter,
  buildIssueFilterBySearchParams,
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractProjectResourceName,
  getSemanticIssueStatusFromSearchParams,
  mergeSearchParams,
  useDynamicLocalStorage,
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

const storedStatus = useDynamicLocalStorage<SemanticIssueStatus>(
  computed(() => `bb.project.issue-list-status.${me.value.name}`),
  "OPEN",
  window.localStorage,
  {
    serializer: {
      read(raw: SemanticIssueStatus) {
        if (!["OPEN", "CLOSED"].includes(raw)) return "OPEN";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  } as UseStorageOptions<SemanticIssueStatus>
);

const defaultSearchParams = (): SearchParams => {
  const myEmail = me.value.email;
  return {
    query: "",
    scopes: [
      ...readonlyScopes.value,
      { id: "status", value: "OPEN" },
      { id: "approval", value: "pending" },
      { id: "approver", value: myEmail },
    ],
  };
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(state.params);
});
const mergedUIIssueFilter = computed(() => {
  return buildUIIssueFilterBySearchParams(state.params);
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

watch(
  () => props.project.name,
  () => {
    state.params = defaultSearchParams();
  }
);

watch(
  () => getSemanticIssueStatusFromSearchParams(state.params),
  (status) => {
    storedStatus.value = status;
  }
);

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
    // Only add readonly scopes (project), don't merge with default preset
    const readonlyParams: SearchParams = {
      query: "",
      scopes: [...readonlyScopes.value],
    };
    state.params = mergeSearchParams(readonlyParams, urlParams);
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
