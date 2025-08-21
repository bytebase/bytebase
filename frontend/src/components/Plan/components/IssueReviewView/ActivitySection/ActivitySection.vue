<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-base font-medium">
        {{ $t("common.activity") }}
      </h3>
    </div>
    <!-- Alert about issue approved and waiting for rollout -->
    <NAlert v-if="showApprovedAlert" type="success" :size="'small'" closable>
      <template #icon>
        <ThumbsUpIcon :size="16" />
      </template>
      <router-link
        v-if="showApprovedAlert && rolloutLink"
        :to="rolloutLink"
        class="inline-flex items-center gap-1 hover:opacity-80"
      >
        <span>{{ $t("issue.approval.approved-and-waiting-for-rollout") }}</span>
        <ExternalLinkIcon class="opacity-80" :size="16" />
      </router-link>
    </NAlert>
    <IssueCommentList />
  </div>
</template>

<script setup lang="ts">
import { ThumbsUpIcon, ExternalLinkIcon } from "lucide-vue-next";
import { NAlert } from "naive-ui";
import { computed } from "vue";
import IssueCommentList from "@/components/IssueV1/components/IssueCommentSection/IssueCommentList.vue";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { extractProjectResourceName } from "@/utils";
import { usePlanContext, useIssueReviewContext } from "../../../logic";

const { rollout, issue } = usePlanContext();
const { project } = useCurrentProjectV1();
const reviewContext = useIssueReviewContext();

// Show alert when issue is approved and rollout hasn't started yet
const showApprovedAlert = computed(() => {
  if (!issue?.value || !rollout?.value) return false;

  // Check if issue is open
  if (issue.value.status !== IssueStatus.OPEN) return false;

  // Check if issue is approved
  const isApproved = reviewContext.done.value;
  if (!isApproved) return false;

  // Check if rollout has started
  const hasStartedTasks = rollout.value.stages.some((stage) =>
    stage.tasks.some(
      (task) =>
        task.status !== Task_Status.NOT_STARTED &&
        task.status !== Task_Status.STATUS_UNSPECIFIED
    )
  );

  return !hasStartedTasks;
});

// Computed property for rollout link
const rolloutLink = computed(() => {
  if (!rollout?.value) return null;

  const rolloutId = rollout.value.name.split("/").pop();
  if (!rolloutId) return null;

  return {
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId,
    },
  };
});
</script>
