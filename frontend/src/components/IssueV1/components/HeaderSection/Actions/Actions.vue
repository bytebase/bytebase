<template>
  <div class="issue-debug">
    <pre>actionType: {{ actionType }}</pre>
    <pre>reviewDone: {{ reviewDone }}</pre>
  </div>
  <div class="flex flex-col items-end">
    <CreateButton v-if="actionType === 'CREATE'" />

    <TinySQLEditorButton v-if="actionType === 'SQL-EDITOR'" />

    <IssueReviewButtonGroup v-if="actionType === 'REVIEW'" />

    <CombinedRolloutButtonGroup v-if="actionType === 'ROLLOUT'" />
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { PresetRoleType } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { isGrantRequestIssue } from "@/utils";
import { useIssueContext } from "../../../logic";
import { CreateButton } from "./create";
import { TinySQLEditorButton } from "./request";
import { IssueReviewButtonGroup } from "./review";
import { CombinedRolloutButtonGroup } from "./rollout";

type ActionType = "CREATE" | "SQL-EDITOR" | "REVIEW" | "ROLLOUT";

const currentUser = useCurrentUserV1();
const { isCreating, issue, reviewContext } = useIssueContext();
const { done: reviewDone } = reviewContext;

const isFinishedGrantRequestIssueByCurrentUser = computed(() => {
  if (isCreating.value) return false;
  if (issue.value.status !== IssueStatus.DONE) return false;
  if (!isGrantRequestIssue(issue.value)) return false;

  return issue.value.creatorEntity.name === currentUser.value.name;
});

const actionType = asyncComputed(async (): Promise<ActionType | undefined> => {
  if (isCreating.value) {
    return "CREATE";
  }

  if (isGrantRequestIssue(issue.value)) {
    if (isFinishedGrantRequestIssueByCurrentUser.value) {
      const role = issue.value.grantRequest?.role;
      if (role === PresetRoleType.SQL_EDITOR_USER) {
        return "SQL-EDITOR";
      }
    }
    return "REVIEW";
  }

  if (reviewDone.value) {
    return "ROLLOUT";
  }

  return "REVIEW";
}, undefined);
</script>
