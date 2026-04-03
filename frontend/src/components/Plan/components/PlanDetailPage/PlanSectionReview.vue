<template>
  <div class="flex flex-col">
    <!-- State: Checking (generating approval flow) -->
    <div
      v-if="issue.approvalStatus === Issue_ApprovalStatus.CHECKING"
      class="flex items-center gap-x-2 text-sm text-control-placeholder p-4"
    >
      <BBSpin :size="16" />
      <span>{{ $t("custom-approval.issue-review.generating-approval-flow") }}</span>
    </div>

    <!-- State: Skipped (no approval required) -->
    <div
      v-else-if="issue.approvalStatus === Issue_ApprovalStatus.SKIPPED || approvalSteps.length === 0"
      class="text-sm text-control-placeholder p-4"
    >
      {{ $t("custom-approval.approval-flow.skip") }}
    </div>

    <!-- State: Has approval flow -->
    <template v-else>
      <!-- Rejection banner -->
      <NAlert
        v-if="issue.approvalStatus === Issue_ApprovalStatus.REJECTED && lastRejection"
        type="warning"
        size="small"
        class="mx-4 mt-3"
        :title="`${$t('custom-approval.issue-review.rejected-by')} ${lastRejection.creator}`"
      >
        <span v-if="lastRejection.comment">{{ lastRejection.comment }}</span>
      </NAlert>

      <!-- Approval steps timeline -->
      <div class="px-4 py-3">
        <h3 class="textinfolabel">{{ $t("issue.approval-flow.self") }}</h3>
        <NTimeline size="large" class="pl-1 pt-3">
          <ApprovalStepItem
            v-for="(step, index) in approvalSteps"
            :key="index"
            :step="step"
            :step-index="index"
            :step-number="index + 1"
            :issue="issue"
          />
        </NTimeline>
      </div>

      <!-- Footer: rule + actions -->
      <div class="flex items-center gap-x-2 px-4 py-2.5 border-t text-sm text-control-placeholder">
        <RiskLevelIcon
          :risk-level="issue.riskLevel"
          :title="approvalTemplate?.title?.trim()"
          class="shrink-0"
        />
        <span v-if="riskLevelText">{{ riskLevelText }}</span>
        <span v-if="ruleText && riskLevelText">·</span>
        <span v-if="ruleText" class="truncate min-w-0">{{ ruleText }}</span>
        <div class="flex-1" />
        <RouterLink
          :to="issueRoute"
          class="text-accent hover:underline"
          active-class=""
          exact-active-class=""
        >
          {{ $t("common.issue") }} #{{ issueUID }} ·
          {{ $t("plan.view-discussion") }} →
        </RouterLink>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NAlert, NTimeline } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import { BBSpin } from "@/bbkit";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  getIssueCommentType,
  IssueCommentType,
  useIssueCommentStore,
} from "@/store/modules/v1/issueComment";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  ListIssueCommentsRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { extractIssueUID, extractProjectResourceName } from "@/utils";
import { usePlanContextWithIssue } from "../../logic/context";
import ApprovalStepItem from "../IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalStepItem.vue";
import RiskLevelIcon from "../IssueReviewView/Sidebar/ApprovalFlowSection/RiskLevelIcon.vue";

const { t } = useI18n();
const { plan, issue } = usePlanContextWithIssue();
const issueCommentStore = useIssueCommentStore();

// Approval data
const approvalTemplate = computed(() => issue.value.approvalTemplate);
const approvalSteps = computed(() => approvalTemplate.value?.flow?.roles ?? []);

const issueUID = computed(() => extractIssueUID(issue.value.name));
const issueRoute = computed(() => ({
  name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
  params: {
    projectId: extractProjectResourceName(plan.value.name),
    issueId: issueUID.value,
  },
}));

const riskLevelText = computed(() => {
  switch (issue.value.riskLevel) {
    case RiskLevel.LOW:
      return t("issue.risk-level.low");
    case RiskLevel.MODERATE:
      return t("issue.risk-level.moderate");
    case RiskLevel.HIGH:
      return t("issue.risk-level.high");
    default:
      return "";
  }
});

const ruleText = computed(() => {
  const parts: string[] = [];
  const title = approvalTemplate.value?.title?.trim();
  const desc = approvalTemplate.value?.description?.trim();
  if (title) parts.push(title);
  if (desc) parts.push(desc);
  return parts.join(" — ");
});

// Fetch comments
watchEffect(async () => {
  if (!issue.value?.name) return;
  try {
    await issueCommentStore.listIssueComments(
      create(ListIssueCommentsRequestSchema, {
        parent: issue.value.name,
        pageSize: 100,
      })
    );
  } catch {
    // Ignore — comments are non-critical
  }
});

const comments = computed(
  () => issueCommentStore.getIssueComments(issue.value.name) ?? []
);

const lastRejection = computed(() => {
  if (issue.value.approvalStatus !== Issue_ApprovalStatus.REJECTED) {
    return undefined;
  }
  for (let i = comments.value.length - 1; i >= 0; i--) {
    const comment = comments.value[i];
    if (getIssueCommentType(comment) === IssueCommentType.APPROVAL) {
      const approval = comment.event?.value;
      if (approval && "status" in approval) {
        return { creator: comment.creator, comment: comment.comment || "" };
      }
    }
  }
  return undefined;
});
</script>
