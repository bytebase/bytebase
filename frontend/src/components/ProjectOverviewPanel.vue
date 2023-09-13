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
      <div class="flex items-center gap-x-1">
        <p class="text-lg font-medium leading-7 text-main">
          {{ $t("common.issue") }}
        </p>
        <button
          type="button"
          class="p-1 rounded hover:bg-gray-200"
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
          <heroicons-outline:search class="h-4 w-4 text-control" />
        </button>
      </div>

      <div>
        <WaitingForMyApprovalIssueTableV1
          v-if="hasCustomApprovalFeature"
          session-key="project-waiting-approval"
          method="LIST"
          :issue-filter="{
            ...commonIssueFilter,
            statusList: [IssueStatus.OPEN],
          }"
        >
          <template #table="{ issueList, loading }">
            <IssueTableV1
              :mode="'PROJECT'"
              :show-placeholder="!loading"
              :title="$t('issue.waiting-approval')"
              :issue-list="issueList"
            />
          </template>
        </WaitingForMyApprovalIssueTableV1>

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
              :title="$t('project.overview.in-progress')"
              :issue-list="issueList"
              :show-placeholder="!loading"
            />
          </template>
        </PagedIssueTableV1>

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
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, computed } from "vue";
import { useRouter } from "vue-router";
import { featureToRef } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Project } from "@/types/proto/v1/project_service";
import { IssueFilter } from "../types";
import IssueTableV1 from "./IssueV1/components/IssueTableV1.vue";
import PagedIssueTableV1 from "./IssueV1/components/PagedIssueTableV1.vue";
import WaitingForMyApprovalIssueTableV1 from "./IssueV1/components/WaitingForMyApprovalIssueTableV1.vue";

interface LocalState {
  isFetchingActivityList: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const state = reactive<LocalState>({
  isFetchingActivityList: false,
});
const router = useRouter();

const hasCustomApprovalFeature = featureToRef("bb.feature.custom-approval");

const commonIssueFilter = computed((): IssueFilter => {
  return {
    project: props.project.name,
    query: "",
  };
});
</script>
