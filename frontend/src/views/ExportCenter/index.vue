<template>
  <div class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <IssueSearch
          v-model:params="state.params"
          class="flex-1"
          :readonly-scopes="readonlyScopes"
          :override-scope-id-list="overrideSearchScopeIdList"
        >
          <template #searchbox-suffix>
            <NTooltip v-if="!disableDataExport" :disabled="allowExportData">
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
            <div v-else />
          </template>
        </IssueSearch>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedIssueTableV1
        :session-key="'export-center'"
        :issue-filter="mergedIssueFilter"
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
      :project-name="specificProject.name"
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
import {
  useCurrentUserV1,
  useProjectV1Store,
  usePolicyByParentAndType,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type } from "@/types/proto/v1/issue_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import {
  buildIssueFilterBySearchParams,
  extractProjectResourceName,
  hasPermissionToCreateDataExportIssueInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";
import DataExportIssueDataTable from "./DataExportIssueDataTable";

const props = defineProps<{
  projectId: string;
}>();

interface LocalState {
  showRequestExportPanel: boolean;
  params: SearchParams;
}

const specificProject = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const exportDataPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_EXPORT,
  }))
);

const readonlyScopes = computed((): SearchScope[] => {
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
  showRequestExportPanel: false,
  params: defaultSearchParams(),
});

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
    "label",
  ];
  return defaultScopeIdList;
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(dataExportIssueSearchParams.value, {
    type: Issue_Type.DATABASE_DATA_EXPORT,
  });
});

const disableDataExport = computed(
  () => exportDataPolicy.value?.exportDataPolicy?.disable ?? false
);

const allowExportData = computed(() => {
  if (specificProject.value) {
    return hasPermissionToCreateDataExportIssueInProject(specificProject.value);
  }

  return projectV1Store.projectList.some((project) =>
    hasPermissionToCreateDataExportIssueInProject(project)
  );
});
</script>
