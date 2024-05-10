<template>
  <div class="w-full px-4">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <IssueSearch
          v-model:params="state.params"
          class="flex-1"
          :readonly-scopes="readonlyScopes"
          :override-scope-id-list="overideSearchScopeIdList"
        >
          <template #searchbox-suffix>
            <NTooltip :disabled="allowExportData">
              <template #trigger>
                <NButton
                  type="primary"
                  :disabled="!allowExportData"
                  @click="state.showRequestExportPanel = true"
                >
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
      <PagedIssueTableV1
        v-model:loading="state.loading"
        v-model:loading-more="state.loadingMore"
        :session-key="'export-center'"
        :issue-filter="mergedIssueFilter"
        :ui-issue-filter="mergedUIIssueFilter"
        :page-size="50"
        :compose-issue-config="{ withRollout: true }"
      >
        <template #table="{ issueList, loading }">
          <DataExportIssueDataTable
            :loading="loading"
            :issue-list="issueList"
            :highlight-text="state.params.query"
            :show-project="!specificProject"
          />
        </template>
      </PagedIssueTableV1>
    </div>
  </div>

  <Drawer
    :auto-focus="true"
    :show="state.showRequestExportPanel"
    @close="state.showRequestExportPanel = false"
  >
    <DataExportPrepForm
      :project-id="specificProject?.uid"
      @dismiss="state.showRequestExportPanel = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import DataExportPrepForm from "@/components/DataExportPrepForm";
import IssueSearch from "@/components/IssueV1/components/IssueSearch/IssueSearch.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { Drawer } from "@/components/v2";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  buildIssueFilterBySearchParams,
  buildUIIssueFilterBySearchParams,
  extractProjectResourceName,
  hasPermissionToCreateDataExportIssueInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";
import DataExportIssueDataTable from "./DataExportIssueDataTable";
import type { ExportRecord } from "./types";

const props = defineProps<{
  projectId?: string;
}>();

interface LocalState {
  exportRecords: ExportRecord[];
  showRequestExportPanel: boolean;
  params: SearchParams;
  loading: boolean;
  loadingMore: boolean;
}

const specificProject = computed(() => {
  return props.projectId
    ? projectV1Store.getProjectByName(`${projectNamePrefix}${props.projectId}`)
    : undefined;
});

const readonlyScopes = computed((): SearchScope[] => {
  if (!specificProject.value) {
    return [];
  }
  return [
    {
      id: "project",
      value: extractProjectResourceName(specificProject.value.name),
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
const projectV1Store = useProjectV1Store();
const state = reactive<LocalState>({
  exportRecords: [],
  showRequestExportPanel: false,
  params: defaultSearchParams(),
  loading: false,
  loadingMore: false,
});

const dataExportIssueSearchParams = computed(() => {
  // Default scopes with type and creator.
  const defaultScopes = [
    {
      id: "type",
      value: "DATA_EXPORT",
    },
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

const overideSearchScopeIdList = computed(() => {
  const defaultScopeIdList: SearchScopeId[] = [
    "status",
    "instance",
    "database",
  ];
  if (!specificProject.value) {
    defaultScopeIdList.push("project");
  }
  return defaultScopeIdList;
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(dataExportIssueSearchParams.value);
});

const mergedUIIssueFilter = computed(() => {
  return buildUIIssueFilterBySearchParams(dataExportIssueSearchParams.value);
});

const allowExportData = computed(() => {
  if (specificProject.value) {
    return hasPermissionToCreateDataExportIssueInProject(
      specificProject.value,
      currentUser.value
    );
  }

  return projectV1Store.projectList.some((project) =>
    hasPermissionToCreateDataExportIssueInProject(project, currentUser.value)
  );
});
</script>
