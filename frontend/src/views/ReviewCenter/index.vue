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
            <NTooltip :disabled="allowToCreateReviewIssue">
              <template #trigger>
                <NButton
                  type="primary"
                  :disabled="!allowToCreateReviewIssue"
                  @click="state.showSQLReviewPanel = true"
                >
                  {{ $t("review-center.review-sql") }}
                </NButton>
              </template>
              {{ $t("common.permission-denied") }}
            </NTooltip>
          </template>
        </IssueSearch>
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedIssueTableV1
        v-model:loading="state.loading"
        v-model:loading-more="state.loadingMore"
        :session-key="'review-center'"
        :issue-filter="mergedIssueFilter"
        :page-size="50"
        :compose-issue-config="{ withPlan: true }"
      >
        <template #table="{ issueList, loading }">
          <SQLReviewIssueDataTable
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
    :show="state.showSQLReviewPanel"
    @close="state.showSQLReviewPanel = false"
  >
    <ReviewIssuePrepForm
      :project-id="specificProject?.uid"
      @dismiss="state.showSQLReviewPanel = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import IssueSearch from "@/components/IssueV1/components/IssueSearch/IssueSearch.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import ReviewIssuePrepForm from "@/components/ReviewIssuePrepForm";
import { Drawer } from "@/components/v2";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type } from "@/types/proto/v1/issue_service";
import {
  buildIssueFilterBySearchParams,
  extractProjectResourceName,
  hasPermissionToCreateReviewIssueInProject,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";
import SQLReviewIssueDataTable from "./SQLReviewIssueDataTable";

const props = defineProps<{
  projectId?: string;
}>();

interface LocalState {
  showSQLReviewPanel: boolean;
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
  showSQLReviewPanel: false,
  params: defaultSearchParams(),
  loading: false,
  loadingMore: false,
});

const issueSearchParams = computed(() => {
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

const overideSearchScopeIdList = computed(() => {
  const defaultScopeIdList: SearchScopeId[] = [
    "status",
    "instance",
    "database",
    "label",
  ];
  if (!specificProject.value) {
    defaultScopeIdList.push("project");
  }
  return defaultScopeIdList;
});

const mergedIssueFilter = computed(() => {
  return buildIssueFilterBySearchParams(issueSearchParams.value, {
    type: Issue_Type.DATABASE_CHANGE,
    hasPipeline: false,
  });
});

const allowToCreateReviewIssue = computed(() => {
  if (specificProject.value) {
    return hasPermissionToCreateReviewIssueInProject(
      specificProject.value,
      currentUser.value
    );
  }

  return projectV1Store.projectList.some((project) =>
    hasPermissionToCreateReviewIssueInProject(project, currentUser.value)
  );
});
</script>
