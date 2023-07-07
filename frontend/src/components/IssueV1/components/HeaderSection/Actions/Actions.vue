<template>
  <div class="flex items-center justify-end gap-x-3">
    <div class="issue-debug">
      <div>actionType: {{ actionType }}</div>
    </div>

    <NButton v-if="actionType === 'CREATE'" type="primary" size="large">
      {{ $t("common.create") }}
    </NButton>

    <ExportCenterButton v-if="actionType === 'EXPORT-CENTER'" />

    <SQLEditorButton v-if="actionType === 'SQL-EDITOR'" />

    <IssueReviewButtonGroup v-if="actionType === 'REVIEW'" />

    <CombinedRolloutButtonGroup v-if="actionType === 'ROLLOUT'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NButton } from "naive-ui";

import { IssueStatus } from "@/types/proto/v1/issue_service";
import { useCurrentUserV1 } from "@/store";
import { isGrantRequestIssue } from "@/utils";
import { useIssueContext } from "../../../logic";
import { IssueReviewButtonGroup } from "./review";
import { CombinedRolloutButtonGroup } from "./rollout";
import { ExportCenterButton, SQLEditorButton } from "./request";

type ActionType =
  | "CREATE"
  | "EXPORT-CENTER"
  | "SQL-EDITOR"
  | "REVIEW"
  | "ROLLOUT";

const currentUser = useCurrentUserV1();
const { isCreating, issue, reviewContext } = useIssueContext();
const { done: reviewDone } = reviewContext;

const isFinishedGrantRequestIssueByCurrentUser = computed(() => {
  if (isCreating.value) return false;
  if (issue.value.status !== IssueStatus.DONE) return false;
  if (!isGrantRequestIssue(issue.value)) return false;

  if (issue.value.creator !== currentUser.value.name) {
    return false;
  }
  return true;
});

const actionType = computed((): ActionType => {
  if (isCreating.value) {
    return "CREATE";
  }
  if (isGrantRequestIssue(issue.value)) {
    if (isFinishedGrantRequestIssueByCurrentUser.value) {
      if (false) {
        // TODO: check request export payload
        // return issue.value.pa.payload.grantRequest?.role === PresetRoleType.EXPORTER;
        return "EXPORT-CENTER";
      }
      if (false) {
        // TODO: check request query payload
        // return issue.value.payload.grantRequest?.role === PresetRoleType.QUERIER;
        return "SQL-EDITOR";
      }
    }
    return "REVIEW";
  }

  return reviewDone.value ? "ROLLOUT" : "REVIEW";
});
</script>
