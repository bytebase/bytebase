<template>
  <div class="flex flex-col gap-y-2">
    <IssueSearch
      v-model:params="state.params"
      :components="['searchbox', 'time-range', 'presets']"
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
import { reactive, computed, watch, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useCurrentUserV1 } from "@/store";
import { useIssueV1Store, useRefreshIssueList } from "@/store";
import type { ComposedIssue } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { SearchParams, SearchScope, SemanticIssueStatus } from "@/utils";
import {
  buildIssueFilterBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractProjectResourceName,
  getSemanticIssueStatusFromSearchParams,
  useDynamicLocalStorage,
  applyUIIssueFilter,
} from "@/utils";
import { IssueSearch } from "../IssueV1/components";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  project: Project;
}>();

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
  return {
    query: "",
    scopes: [
      ...readonlyScopes.value,
      { id: "status", value: storedStatus.value },
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
</script>
