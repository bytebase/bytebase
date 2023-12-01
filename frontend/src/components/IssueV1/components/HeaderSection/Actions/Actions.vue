<template>
  <div class="flex flex-col items-end">
    <CreateButton v-if="actionType === 'CREATE'" />

    <ExportCenterButton v-if="actionType === 'EXPORT-CENTER'" />

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
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueContext } from "../../../logic";
import { CreateButton } from "./create";
import { ExportCenterButton, TinySQLEditorButton } from "./request";
import { IssueReviewButtonGroup } from "./review";
import { CombinedRolloutButtonGroup } from "./rollout";

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

  return issue.value.creatorEntity.name === currentUser.value.name;
});

const actionType = asyncComputed(async (): Promise<ActionType | undefined> => {
  if (isCreating.value) {
    return "CREATE";
  }
  if (isGrantRequestIssue(issue.value)) {
    if (isFinishedGrantRequestIssueByCurrentUser.value) {
      const role = issue.value.grantRequest?.role;
      if (role === PresetRoleType.EXPORTER) {
        // Show the export button only when the grant request condition is based on the statement.
        const expr = await convertFromCELString(
          issue.value.grantRequest?.condition?.expression ?? ""
        );
        if (expr.statement) {
          return "EXPORT-CENTER";
        }
      }
      if (role === PresetRoleType.QUERIER) {
        return "SQL-EDITOR";
      }
    }
    return "REVIEW";
  }

  return reviewDone.value ? "ROLLOUT" : "REVIEW";
}, undefined);
</script>
