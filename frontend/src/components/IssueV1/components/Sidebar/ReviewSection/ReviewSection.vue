<template>
  <div v-if="!isCreating" class="flex flex-col gap-y-2">
    <div class="w-full flex flex-row items-center gap-2">
      <NTooltip placement="bottom">
        <template #trigger>
          <div>
            <div class="textlabel flex items-center gap-x-1">
              {{ $t("issue.approval-flow.self") }}
              <FeatureBadge :feature="PlanLimitConfig_Feature.APPROVAL_WORKFLOW" />
            </div>
          </div>
        </template>
        <template #default>
          <div class="max-w-[22rem]">
            {{ $t("issue.approval-flow.tooltip") }}
          </div>
        </template>
      </NTooltip>

      <RiskLevelTag />
    </div>

    <div class="flex-1 min-w-[14rem]">
      <div
        v-if="!ready"
        class="flex items-center gap-x-2 text-sm text-control-placeholder"
      >
        <BBSpin :size="20" />
        <span>
          {{ $t("custom-approval.issue-review.generating-approval-flow") }}
        </span>
      </div>
      <div v-else-if="error" class="flex items-center gap-x-2">
        <NTooltip>
          <template #trigger>
            <span class="text-error text-sm">{{ $t("common.error") }}</span>
          </template>

          <div class="max-w-[20rem]">{{ error }}</div>
        </NTooltip>
        <NButton
          size="tiny"
          :loading="retrying"
          @click="retryFindingApprovalFlow"
        >
          {{ $t("common.retry") }}
        </NButton>
      </div>
      <Timeline
        v-else-if="wrappedSteps.length > 0"
        :steps="wrappedSteps"
        class="mt-1"
      />
      <div
        v-else
        class="flex items-center text-sm text-control-placeholder gap-x-1"
      >
        {{ $t("custom-approval.approval-flow.skip") }}
      </div>
    </div>
  </div>
  <div v-if="isCreating" class="flex flex-col gap-y-1">
    <div class="textlabel flex items-center gap-x-1">
      {{ $t("issue.approval-flow.self") }}
      <FeatureBadge :feature="PlanLimitConfig_Feature.APPROVAL_WORKFLOW" />
    </div>
    <div class="text-control-placeholder text-xs">
      {{ $t("issue.approval-flow.pre-issue-created-tips") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NTooltip } from "naive-ui";
import { ref } from "vue";
import { BBSpin } from "@/bbkit";
import FeatureBadge from "@/components/FeatureGuard/FeatureBadge.vue";
import { useIssueContext, useWrappedReviewStepsV1 } from "@/components/IssueV1";
import { useIssueV1Store } from "@/store";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import RiskLevelTag from "./RiskLevelTag.vue";
import Timeline from "./Timeline.vue";

const { issue, events, isCreating, reviewContext } = useIssueContext();
const { ready, error } = reviewContext;

const wrappedSteps = useWrappedReviewStepsV1(issue, reviewContext);

const retrying = ref(false);
const retryFindingApprovalFlow = async () => {
  retrying.value = true;
  try {
    await useIssueV1Store().regenerateReviewV1(issue.value.name);
    events.emit("status-changed", { eager: true });
  } finally {
    retrying.value = false;
  }
};
</script>
