<template>
  <div class="flex items-stretch gap-x-3">
    <ReviewActionButton
      v-for="action in issueReviewActionList"
      :key="action"
      :action="action"
      @perform-action="
        (action) => events.emit('perform-issue-review-action', { action })
      "
    />

    <IssueStatusActionButtonGroup
      :display-mode="displayMode"
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="forceRolloutActionList"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { IssueReviewAction } from "@/components/IssueV1";
import {
  getApplicableIssueStatusActionList,
  getApplicableStageRolloutActionList,
  getApplicableTaskRolloutActionList,
  taskRolloutActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1";
import { extractUserId, useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isDatabaseChangeRelatedIssue,
} from "@/utils";
import type { ExtraActionOption } from "../common";
import { IssueStatusActionButtonGroup } from "../common";
import ReviewActionButton from "./ReviewActionButton.vue";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { issue, isCreating, events, selectedTask, selectedStage } =
  useIssueContext();
const { project } = useCurrentProjectV1();

const shouldShowApproveOrReject = computed(() => {
  if (
    issue.value.status === IssueStatus.CANCELED ||
    issue.value.status === IssueStatus.DONE ||
    isCreating.value
  ) {
    return false;
  }

  // Must be pending approval
  if (issue.value.approvalStatus !== Issue_ApprovalStatus.PENDING) {
    return false;
  }

  // Hide review actions if self-approval is disabled
  if (
    !project.value.allowSelfApproval &&
    currentUser.value.email === extractUserId(issue.value.creator)
  ) {
    return false;
  }

  return true;
});

const shouldShowReRequestReview = computed(() => {
  return (
    extractUserId(issue.value.creator) === currentUser.value.email &&
    issue.value.approvalStatus === Issue_ApprovalStatus.REJECTED
  );
});

const issueReviewActionList = computed(() => {
  const actionList: IssueReviewAction[] = [];
  if (shouldShowApproveOrReject.value) {
    actionList.push("SEND_BACK", "APPROVE");
  }
  if (shouldShowReRequestReview.value) {
    actionList.push("RE_REQUEST");
  }
  return actionList;
});

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});
const forceRolloutActionList = computed((): ExtraActionOption[] => {
  // If it's not a database change related issue, no force rollout actions.
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return [];
  }

  if (
    !hasWorkspacePermissionV2("bb.taskRuns.create") &&
    !hasProjectPermissionV2(project.value, "bb.taskRuns.create")
  ) {
    // Only for users with permission to create task runs.
    return [];
  }

  const taskExtraActionOptions = getApplicableTaskRolloutActionList(
    issue.value,
    selectedTask.value
  ).map<ExtraActionOption>((action) => ({
    type: "TASK",
    action,
    target: selectedTask.value,
    label: `${t("common.force-verb", {
      verb: taskRolloutActionDisplayName(action).toLowerCase(),
    })}`,
    key: `force-${action}-task`,
  }));
  const stageExtraActionOptions = getApplicableStageRolloutActionList(
    issue.value,
    selectedStage.value
  ).map<ExtraActionOption>(({ action, tasks }) => {
    const taskActionName = taskRolloutActionDisplayName(action);
    const stageActionName = t("issue.action-to-current-stage", {
      action: taskActionName,
    });
    return {
      type: "TASK-BATCH",
      action,
      target: tasks,
      label: t("common.force-verb", { verb: stageActionName.toLowerCase() }),
      key: `force-${action}-stage`,
    };
  });

  return [...taskExtraActionOptions, ...stageExtraActionOptions];
});

const displayMode = computed(() => {
  return issueReviewActionList.value.length > 0 ? "DROPDOWN" : "BUTTON";
});
</script>
