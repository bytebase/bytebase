<template>
  <div class="space-y-6">
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("project.overview.recent-activity") }}
      </p>
      <div class="relative">
        <!-- show the first 5 activities -->
        <!-- But won't show "Load more", since we have a "View all" link below -->
        <PagedActivityTable
          :activity-find="{
            resource: project.name,
            order: 'desc',
          }"
          session-key="bb.page-activity-table.project-activity"
          :page-size="5"
          :hide-load-more="true"
        >
          <template #table="{ list }">
            <ActivityTable :activity-list="list" />
          </template>
        </PagedActivityTable>
        <div
          v-if="state.isFetchingActivityList"
          class="absolute inset-0 flex flex-col items-center justify-center bg-white/70"
        >
          <BBSpin />
        </div>
        <div class="w-full flex justify-end mt-2 px-4">
          <router-link
            :to="`#activity`"
            class="normal-link"
            exact-active-class=""
          >
            {{ $t("project.overview.view-all") }}
          </router-link>
        </div>
      </div>
    </div>

    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.issue") }}
      </p>

      <div>
        <WaitingForMyApprovalIssueTable
          v-if="hasCustomApprovalFeature"
          session-key="project-waiting-approval"
          :issue-find="{
            statusList: ['OPEN'],
            projectId: project.uid,
          }"
        >
          <template #table="{ issueList, loading }">
            <IssueTable
              :mode="'PROJECT'"
              :show-placeholder="!loading"
              :title="$t('issue.waiting-approval')"
              :issue-list="issueList"
            />
          </template>
        </WaitingForMyApprovalIssueTable>

        <!-- show OPEN issues with pageSize=10 -->
        <PagedIssueTable
          session-key="project-open"
          :issue-find="{
            statusList: ['OPEN'],
            projectId: project.uid,
          }"
          :page-size="10"
        >
          <template #table="{ issueList, loading }">
            <IssueTable
              class="-mt-px"
              :mode="'PROJECT'"
              :title="$t('project.overview.in-progress')"
              :issue-list="issueList"
              :show-placeholder="!loading"
            />
          </template>
        </PagedIssueTable>

        <!-- show the first 5 DONE or CANCELED issues -->
        <!-- But won't show "Load more", since we have a "View all closed" link below -->
        <PagedIssueTable
          session-key="project-closed"
          :issue-find="{
            statusList: ['DONE', 'CANCELED'],
            projectId: project.uid,
          }"
          :page-size="5"
          :hide-load-more="true"
        >
          <template #table="{ issueList, loading }">
            <IssueTable
              class="-mt-px"
              :mode="'PROJECT'"
              :title="$t('project.overview.recently-closed')"
              :issue-list="issueList"
              :show-placeholder="!loading"
            />
          </template>
        </PagedIssueTable>

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
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType } from "vue";
import PagedIssueTable from "@/components/Issue/table/PagedIssueTable.vue";
import { featureToRef } from "@/store";
import { Project } from "@/types/proto/v1/project_service";
import { IssueTable } from "../components/Issue";
import { Issue } from "../types";

interface LocalState {
  isFetchingActivityList: boolean;
  progressIssueList: Issue[];
  closedIssueList: Issue[];
}

defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const state = reactive<LocalState>({
  isFetchingActivityList: false,
  progressIssueList: [],
  closedIssueList: [],
});
const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");
</script>
