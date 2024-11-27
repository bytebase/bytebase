<template>
  <div class="flex items-stretch gap-x-3">
    <template v-if="!hideIssueReviewActions">
      <ReviewActionButton
        v-for="action in issueReviewActionList"
        :key="action"
        :action="action"
        @perform-action="
          (action) => events.emit('perform-issue-review-action', { action })
        "
      />
    </template>

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
import { useCurrentUserV1, useAppFeature, extractUserEmail } from "@/store";
import { PresetRoleType } from "@/types";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import {
  extractUserResourceName,
  isDatabaseChangeRelatedIssue,
  hasWorkspaceLevelRole,
} from "@/utils";
import type { ExtraActionOption } from "../common";
import { IssueStatusActionButtonGroup } from "../common";
import ReviewActionButton from "./ReviewActionButton.vue";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const hideIssueReviewActions = useAppFeature(
  "bb.feature.issue.hide-review-actions"
);
const { issue, phase, reviewContext, events, selectedTask, selectedStage } =
  useIssueContext();
const { ready, status, done } = reviewContext;

const shouldShowApproveOrReject = computed(() => {
  if (
    issue.value.status === IssueStatus.CANCELED ||
    issue.value.status === IssueStatus.DONE
  ) {
    return false;
  }

  // Do not show review actions for the creator.
  if (currentUser.value.email === extractUserEmail(issue.value.creator)) {
    return false;
  }

  if (phase.value !== "REVIEW") {
    return false;
  }

  if (!ready.value) return false;
  if (done.value) return false;

  return true;
});
const shouldShowApprove = computed(() => {
  if (!shouldShowApproveOrReject.value) return false;
  return status.value === Issue_Approver_Status.PENDING;
});
const shouldShowReject = computed(() => {
  if (!shouldShowApproveOrReject.value) return false;
  return status.value === Issue_Approver_Status.PENDING;
});
const shouldShowReRequestReview = computed(() => {
  return (
    extractUserResourceName(issue.value.creator) === currentUser.value.email &&
    status.value === Issue_Approver_Status.REJECTED
  );
});
const issueReviewActionList = computed(() => {
  const actionList: IssueReviewAction[] = [];
  if (shouldShowReject.value) {
    actionList.push("SEND_BACK");
  }
  if (shouldShowApprove.value) {
    actionList.push("APPROVE");
  }
  if (shouldShowReRequestReview.value) {
    actionList.push("RE_REQUEST");
  }

  return actionList;
});

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(
    issue.value,
    reviewContext.status.value
  );
});
const forceRolloutActionList = computed((): ExtraActionOption[] => {
  // If it's not a database change related issue, no force rollout actions.
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return [];
  }

  // Still using role based permission checks
  if (
    !hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_ADMIN) &&
    !hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_DBA)
  ) {
    // Only for workspace admins and DBAs.
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
  if (hideIssueReviewActions.value) return "BUTTON";
  return issueReviewActionList.value.length > 0 ? "DROPDOWN" : "BUTTON";
});
</script>
