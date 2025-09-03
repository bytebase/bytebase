<template>
  <div class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <IssueSearch
          v-model:params="state.params"
          class="flex-1"
          :override-scope-id-list="overrideSearchScopeIdList"
        >
          <template #searchbox-suffix>
            <NTooltip :disabled="allowExportData">
              <template #trigger>
                <NButton
                  type="primary"
                  :disabled="!allowExportData"
                  @click="state.showRequestExportPanel = true"
                >
                  <template #icon>
                    <DownloadIcon class="h-4 w-4" />
                  </template>
                  {{ $t("quick-action.request-export-data") }}
                </NButton>
              </template>
              {{ $t("export-center.permission-denied") }}
            </NTooltip>
          </template>
        </IssueSearch>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedTable
        ref="issuePagedTable"
        :session-key="'export-center'"
        :fetch-list="fetchIssueList"
      >
        <template #table="{ list, loading }">
          <IssueTableV1
            :loading="loading"
            :issue-list="list"
            :highlight-text="state.params.query"
          />
        </template>
      </PagedTable>
    </div>
  </div>

  <Drawer
    :auto-focus="true"
    :show="state.showRequestExportPanel"
    @close="state.showRequestExportPanel = false"
  >
    <DataExportPrepForm
      :project-name="specificProject.name"
      @dismiss="state.showRequestExportPanel = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { DownloadIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import DataExportPrepForm from "@/components/DataExportPrepForm";
import IssueSearch from "@/components/IssueV1/components/IssueSearch/IssueSearch.vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import { Drawer } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  useCurrentUserV1,
  useProjectByName,
  useIssueV1Store,
  useRefreshIssueList,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue } from "@/types";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  extractProjectResourceName,
  hasPermissionToCreateDataExportIssueInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

interface LocalState {
  showRequestExportPanel: boolean;
  params: SearchParams;
}

const { project: specificProject } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const readonlyScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      value: extractProjectResourceName(specificProject.value.name),
      readonly: true,
    },
  ];
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [...readonlyScopes.value, { id: "status", value: "OPEN" }],
  };
  return params;
};

const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  showRequestExportPanel: false,
  params: defaultSearchParams(),
});

watch(
  () => props.projectId,
  () => {
    state.params = defaultSearchParams();
  }
);

const issueStore = useIssueV1Store();
const issuePagedTable =
  ref<ComponentExposed<typeof PagedTable<ComposedIssue>>>();

const dataExportIssueSearchParams = computed(() => {
  // Default scopes with type and creator.
  const defaultScopes = [
    {
      id: "creator",
      value: currentUser.value.email,
    },
  ];
  // If specific project is provided, add project scope.
  if (specificProject.value) {
    defaultScopes.push({
      id: "project",
      value: extractProjectResourceName(specificProject.value.name),
    });
  }
  return {
    query: state.params.query,
    scopes: [...state.params.scopes, ...defaultScopes],
  } as SearchParams;
});

const overrideSearchScopeIdList = computed(() => {
  const defaultScopeIdList: SearchScopeId[] = [
    "status",
    "instance",
    "database",
    "issue-label",
  ];
  return defaultScopeIdList;
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(dataExportIssueSearchParams.value, {
    type: Issue_Type.DATABASE_EXPORT,
  });
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

const allowExportData = computed(() => {
  return hasPermissionToCreateDataExportIssueInProject(specificProject.value);
});
</script>
