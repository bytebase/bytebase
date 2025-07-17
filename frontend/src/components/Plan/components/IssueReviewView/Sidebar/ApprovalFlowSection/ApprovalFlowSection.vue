<template>
  <div>
    <div class="w-full flex flex-row items-center gap-2">
      <h3 class="textlabel">{{ $t("issue.approval-flow.self") }}</h3>
      <FeatureBadge :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      <RiskLevelIcon :risk-level="issue.riskLevel" />
    </div>

    <div class="mt-2">
      <div
        v-if="!issue.approvalFindingDone"
        class="flex items-center gap-x-2 text-sm text-control-placeholder"
      >
        <BBSpin :size="16" />
        <span>
          {{ $t("custom-approval.issue-review.generating-approval-flow") }}
        </span>
      </div>
      <div
        v-else-if="issue.approvalFindingError"
        class="flex items-center gap-x-2"
      >
        <span class="text-error text-sm">{{ issue.approvalFindingError }}</span>
        <NButton
          size="tiny"
          :loading="retrying"
          @click="retryFindingApprovalFlow"
        >
          {{ $t("common.retry") }}
        </NButton>
      </div>
      <NTimeline
        v-else-if="approvalSteps.length > 0"
        size="large"
        class="pl-1 mt-1"
      >
        <ApprovalStepItem
          v-for="(step, index) in approvalSteps"
          :key="index"
          :step="step"
          :step-index="index"
          :step-number="index + 1"
          :issue="issue"
        />
      </NTimeline>
      <div
        v-else
        class="flex items-center text-sm text-control-placeholder gap-x-1"
      >
        {{ $t("custom-approval.approval-flow.skip") }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NTimeline, NButton } from "naive-ui";
import { computed, ref } from "vue";
import { BBSpin } from "@/bbkit";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import { useIssueV1Store } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import ApprovalStepItem from "./ApprovalStepItem.vue";
import RiskLevelIcon from "./RiskLevelIcon.vue";

const props = defineProps<{
  issue: Issue;
}>();

const emit = defineEmits<{
  (e: "issue-updated"): void;
}>();

const approvalTemplate = computed(() => {
  const templates = props.issue.approvalTemplates || [];
  return templates.length > 0 ? templates[0] : undefined;
});

const approvalSteps = computed(() => {
  return approvalTemplate.value?.flow?.steps || [];
});

const retrying = ref(false);

const retryFindingApprovalFlow = async () => {
  retrying.value = true;
  try {
    await useIssueV1Store().regenerateReviewV1(props.issue.name);
    emit("issue-updated");
  } finally {
    retrying.value = false;
  }
};
</script>
