<template>
  <div class="space-y-2">
    <div class="flex items-center">
      <div class="flex-1 overflow-hidden">
        <TabFilter v-model:value="state.tab" :items="tabItemList" />
      </div>
      <div class="flex flex-row space-x-4 p-0.5">
        <div class="flex items-center gap-x-1">
          <p class="font-medium leading-7 text-main">
            {{ $t("issue.advanced-search.self") }}
          </p>
          <button
            type="button"
            class="p-1 rounded bg-gray-200 hover:bg-gray-300 border border-gray-300"
            @click="
              () => {
                router.replace({
                  name: 'workspace.issue',
                  query: {
                    project: project.uid,
                    autofocus: 1,
                  },
                });
              }
            "
          >
            <heroicons-outline:search class="h-3.5 w-3.5 text-control" />
          </button>
        </div>
        <SearchBox
          :value="state.searchText"
          :placeholder="$t('common.filter-by-name')"
          :autofocus="true"
          @update:value="changeSearchText($event)"
        />
      </div>
    </div>

    <div v-show="state.tab === 'WAITING_APPROVAL'" class="mt-2">
      <WaitingForMyApprovalIssueTableV1
        v-if="hasCustomApprovalFeature"
        session-key="project-waiting-approval"
        :project="commonIssueFilter.project"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            :mode="'PROJECT'"
            :show-placeholder="!loading"
            :issue-list="issueList.filter(keywordFilter)"
            title=""
          />
        </template>
      </WaitingForMyApprovalIssueTableV1>
    </div>

    <div v-show="state.tab === 'OPEN'" class="mt-2">
      <!-- show OPEN issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="project-open"
        method="LIST"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
        }"
        :page-size="10"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="-mt-px"
            :mode="'PROJECT'"
            :issue-list="issueList.filter(keywordFilter)"
            :show-placeholder="!loading"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="state.tab === 'RECENTLY_CLOSED'" class="mt-2">
      <!-- show the first 5 DONE or CANCELED issues -->
      <!-- But won't show "Load more", since we have a "View all closed" link below -->
      <PagedIssueTableV1
        session-key="project-closed"
        method="LIST"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
        }"
        :page-size="5"
        :hide-load-more="true"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="-mt-px"
            :mode="'PROJECT'"
            :title="$t('project.overview.recently-closed')"
            :issue-list="issueList.filter(keywordFilter)"
            :show-placeholder="!loading"
          />
        </template>
      </PagedIssueTableV1>

      <div class="w-full flex justify-end mt-2 px-4">
        <router-link
          :to="`/issue?status=closed&project=${project.uid}`"
          class="normal-link"
        >
          {{ $t("project.overview.view-all-closed") }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import WaitingForMyApprovalIssueTableV1 from "@/components/IssueV1/components/WaitingForMyApprovalIssueTableV1.vue";
import { TabFilterItem } from "@/components/v2";
import { featureToRef, useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { ComposedIssue, IssueFilter } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Project } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV1 } from "@/utils";

const TABS = ["WAITING_APPROVAL", "OPEN", "RECENTLY_CLOSED"] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  tab: TabValue;
  searchText: string;
  isFetchingActivityList: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const state = reactive<LocalState>({
  tab: "WAITING_APPROVAL",
  searchText: "",
  isFetchingActivityList: false,
});
const { t } = useI18n();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();

const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-issue",
    currentUserV1.value.userRole
  );
});

const tabItemList = computed((): TabFilterItem<TabValue>[] => {
  const WAITING_APPROVAL: TabFilterItem<TabValue> = {
    value: "WAITING_APPROVAL",
    label: t("issue.waiting-approval"),
  };
  const list = hasCustomApprovalFeature.value ? [WAITING_APPROVAL] : [];
  return [
    ...list,
    { value: "OPEN", label: t("project.overview.in-progress") },
    { value: "RECENTLY_CLOSED", label: t("project.overview.recently-closed") },
  ];
});

const commonIssueFilter = computed((): IssueFilter => {
  let principal = "";
  if (!hasPermission.value) {
    principal = `${userNamePrefix}${currentUserV1.value.email}`;
  }
  return {
    project: props.project.name,
    query: "",
    principal,
  };
});

const keywordFilter = (issue: ComposedIssue) => {
  const keyword = state.searchText.trim().toLowerCase();
  if (keyword) {
    if (!issue.title.toLowerCase().includes(keyword)) {
      return false;
    }
  }
  return true;
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};

watch(
  [hasCustomApprovalFeature, () => state.tab],
  () => {
    if (!hasCustomApprovalFeature.value && state.tab === "WAITING_APPROVAL") {
      state.tab = "OPEN";
    }
  },
  { immediate: true }
);
</script>
