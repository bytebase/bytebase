<template>
  <div :key="viewId" class="flex flex-col">
    <IssueSearch
      v-model:params="state.params"
      :components="['searchbox', 'time-range', 'presets', 'filters']"
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
            :issue-list="applyUIIssueFilter(list, mergedUIIssueFilter)"
            :highlight-text="state.params.query"
            :show-project="true"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage, type UseStorageOptions } from "@vueuse/core";
import { reactive, computed, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
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
import type { SearchParams, SemanticIssueStatus } from "@/utils";
import {
  buildIssueFilterBySearchParams,
  buildUIIssueFilterBySearchParams,
  getSemanticIssueStatusFromSearchParams,
  useDynamicLocalStorage,
  applyUIIssueFilter,
} from "@/utils";
import { getComponentIdLocalStorageKey } from "@/utils/localStorage";

interface LocalState {
  params: SearchParams;
}

const me = useCurrentUserV1();
const issueStore = useIssueV1Store();
const issuePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedIssue>>>();

const viewId = useLocalStorage<string>(
  getComponentIdLocalStorageKey(WORKSPACE_ROUTE_MY_ISSUES),
  ""
);

const storedStatus = useDynamicLocalStorage<SemanticIssueStatus>(
  computed(() => `bb.home.issue-list-status.${me.value.name}`),
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
  return {
    query: "",
    scopes: [
      {
        id: "status",
        value: storedStatus.value,
      },
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
  () => getSemanticIssueStatusFromSearchParams(state.params),
  (status) => {
    storedStatus.value = status;
  }
);
</script>
