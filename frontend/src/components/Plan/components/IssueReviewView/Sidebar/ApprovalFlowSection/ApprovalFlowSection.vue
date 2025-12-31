<template>
  <div>
    <div class="w-full flex flex-row items-center gap-2">
      <h3 class="textlabel">{{ $t("issue.approval-flow.self") }}</h3>
      <FeatureBadge :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      <RiskLevelIcon :risk-level="issue.riskLevel" />
    </div>

    <div class="mt-2">
      <div
        v-if="issue.approvalStatus === Issue_ApprovalStatus.CHECKING"
        class="flex items-center gap-x-2 text-sm text-control-placeholder"
      >
        <BBSpin :size="16" />
        <span>
          {{ $t("custom-approval.issue-review.generating-approval-flow") }}
        </span>
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
import { NTimeline } from "naive-ui";
import { computed } from "vue";
import { BBSpin } from "@/bbkit";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import ApprovalStepItem from "./ApprovalStepItem.vue";
import RiskLevelIcon from "./RiskLevelIcon.vue";

const props = defineProps<{
  issue: Issue;
}>();

const approvalTemplate = computed(() => {
  return props.issue.approvalTemplate;
});

const approvalSteps = computed(() => {
  return approvalTemplate.value?.flow?.roles || [];
});
</script>
