<template>
  <div class="space-y-2">
    <div class="flex items-center">
      <div class="flex-1 overflow-hidden">
        <TabFilter v-model:value="state.tab" :items="tabItemList" />
      </div>
      <div class="flex flex-row space-x-4 p-0.5">
        <router-link
          :to="`/issue?project=${project.uid}`"
          class="flex space-x-1 items-center normal-link !whitespace-nowrap"
        >
          <heroicons-outline:search class="h-4 w-4" />
          <span class="hidden md:block">{{
            $t("issue.advanced-search.self")
          }}</span>
        </router-link>
      </div>
    </div>

    <div v-show="state.tab === 'WAITING_APPROVAL'" class="mt-2">
      <PagedIssueTableV1
        v-if="hasCustomApprovalFeature"
        session-key="project-waiting-approval"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
        }"
        :ui-issue-filter="{
          approval: 'pending',
        }"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            :mode="'PROJECT'"
            :show-placeholder="!loading"
            :issue-list="issueList"
            title=""
          />
        </template>
      </PagedIssueTableV1>
    </div>

    <div v-show="state.tab === 'OPEN'" class="mt-2">
      <!-- show OPEN issues with pageSize=10 -->
      <PagedIssueTableV1
        session-key="project-open"
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.OPEN],
        }"
        :page-size="50"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="-mt-px"
            :mode="'PROJECT'"
            :issue-list="issueList"
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
        :issue-filter="{
          ...commonIssueFilter,
          statusList: [IssueStatus.DONE, IssueStatus.CANCELED],
        }"
        :page-size="50"
        :hide-load-more="true"
      >
        <template #table="{ issueList, loading }">
          <IssueTableV1
            class="-mt-px"
            :mode="'PROJECT'"
            :title="$t('project.overview.recently-closed')"
            :issue-list="issueList"
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
import IssueTableV1 from "@/components/IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "@/components/IssueV1/components/PagedIssueTableV1.vue";
import { TabFilterItem } from "@/components/v2";
import { featureToRef, useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { IssueFilter } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Project } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV1 } from "@/utils";

const TABS = ["WAITING_APPROVAL", "OPEN", "RECENTLY_CLOSED"] as const;

type TabValue = typeof TABS[number];

interface LocalState {
  tab: TabValue;
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
  isFetchingActivityList: false,
});
const { t } = useI18n();
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
