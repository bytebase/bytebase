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
            <PermissionGuardWrapper
              v-slot="slotProps"
              :project="specificProject"
              :permissions="['bb.issues.create', 'bb.plans.create', 'bb.rollouts.create']"
            >
              <NButton
                type="primary"
                :disabled="slotProps.disabled"
                @click="state.showRequestExportPanel = true"
              >
                <template #icon>
                  <DownloadIcon class="h-4 w-4" />
                </template>
                {{ $t("quick-action.request-export-data") }}
              </NButton>
            </PermissionGuardWrapper>
          </template>
        </IssueSearch>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-80">
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
import { NButton } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import DataExportPrepForm from "@/components/DataExportPrepForm";
import IssueSearch from "@/components/IssueV1/components/IssueSearch/IssueSearch.vue";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Drawer } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  useIssueV1Store,
  useProjectByName,
  useRefreshIssueList,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue } from "@/types";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  buildIssueFilterBySearchParams,
  extractProjectResourceName,
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
    scopes: [
      ...readonlyScopes.value,
      { id: "status", value: IssueStatus[IssueStatus.OPEN] },
    ],
  };
  return params;
};

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
  // Default scopes with type.
  const defaultScopes = [];
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
</script>
