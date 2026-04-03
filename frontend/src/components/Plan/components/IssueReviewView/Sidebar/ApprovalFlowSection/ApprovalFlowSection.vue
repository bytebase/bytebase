<template>
  <div>
    <div class="w-full flex flex-row items-center gap-2 flex-wrap">
      <h3 class="textinfolabel">{{ $t("issue.approval-flow.self") }}</h3>
      <FeatureBadge :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      <RiskLevelIcon
        :risk-level="issue.riskLevel"
        :title="approvalTemplate?.title?.trim()"
      />
      <div class="grow" />
      <NTag class="truncate" v-if="statusTag" size="small" round :type="statusTag.type">
        {{ statusTag.label }}
      </NTag>
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
import type { TagProps } from "naive-ui";
import { NTag, NTimeline } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import ApprovalStepItem from "./ApprovalStepItem.vue";
import RiskLevelIcon from "./RiskLevelIcon.vue";

interface StatusTag {
  label: string;
  type?: TagProps["type"];
}

const props = defineProps<{
  issue: Issue;
}>();

const { t } = useI18n();

const approvalTemplate = computed(() => {
  return props.issue.approvalTemplate;
});

const approvalSteps = computed(() => {
  return approvalTemplate.value?.flow?.roles || [];
});

const statusTag = computed((): StatusTag | undefined => {
  const status = props.issue.approvalStatus;
  if (approvalSteps.value.length === 0) return undefined;

  if (status === Issue_ApprovalStatus.APPROVED) {
    return {
      label: t("issue.table.approved"),
      type: "success",
    };
  }
  if (status === Issue_ApprovalStatus.REJECTED) {
    return {
      label: t("common.rejected"),
      type: "warning",
    };
  }
  if (status === Issue_ApprovalStatus.PENDING) {
    return {
      label: t("common.under-review"),
      type: "info",
    };
  }
  return undefined;
});
</script>
