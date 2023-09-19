<template>
  <NTimeline :icon-size="20" size="medium" class="bb-approval-timeline">
    <NTimelineItem v-for="step in steps" :key="step.index">
      <template #icon>
        <div
          class="w-5 h-5 rounded-full flex items-center justify-center text-xs shrink-0"
          :class="iconClass(step)"
        >
          <heroicons-outline:thumb-up
            v-if="step.status === 'APPROVED'"
            class="w-3.5 h-3.5 text-white"
          />
          <heroicons:pause-solid
            v-else-if="step.status === 'REJECTED'"
            class="w-3.5 h-3.5 text-white"
          />
          <template v-else-if="step.status === 'CURRENT'">
            <heroicons-outline:external-link
              v-if="isExternalApprovalStep(step.step)"
              class="w-3.5 h-3.5"
            />
            <heroicons-outline:user v-else class="w-3.5 h-3.5" />
          </template>
          <template v-else>
            {{ step.index + 1 }}
          </template>
        </div>
      </template>

      <div class="flex-1 flex text-sm overflow-hidden" :class="itemClass(step)">
        <template v-if="!isExternalApprovalStep(step.step)">
          <div class="whitespace-nowrap shrink-0">
            {{ approvalNodeText(step.step.nodes[0]) }}
          </div>
          <div class="mr-1.5 shrink-0">:</div>
          <div class="flex-1 overflow-hidden">
            <NEllipsis
              v-if="step.status === 'APPROVED'"
              class="inline-block"
              :class="step.approver?.name === currentUser.name && 'font-bold'"
            >
              <span>{{ step.approver?.title }}</span>
              <span v-if="step.approver?.name === currentUser.name">
                ({{ $t("custom-approval.issue-review.you") }})
              </span>
              <span
                v-if="step.approver?.name === USER_SYSTEM_BOT"
                class="ml-2 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
              >
                {{ $t("settings.members.system-bot") }}
              </span>
            </NEllipsis>
            <Candidates v-else :step="step" />
          </div>
        </template>
        <template v-else>
          <div class="whitespace-nowrap shrink-0">
            {{ approvalNodeText(step.step.nodes[0]) }}
          </div>
          <ExternalApprovalSyncButton />
        </template>
      </div>
    </NTimelineItem>
  </NTimeline>
</template>

<script lang="ts" setup>
import { NTimeline, NTimelineItem, NEllipsis } from "naive-ui";
import { storeToRefs } from "pinia";
import { useAuthStore } from "@/store";
import { WrappedReviewStep } from "@/types";
import { ApprovalStep } from "@/types/proto/v1/issue_service";
import { approvalNodeText } from "@/utils";
import Candidates from "./Candidates.vue";
import ExternalApprovalSyncButton from "./ExternalApprovalNodeSyncButton.vue";

const USER_SYSTEM_BOT = "users/1";

defineProps<{
  steps: WrappedReviewStep[];
}>();

const { currentUser } = storeToRefs(useAuthStore());

const isExternalApprovalStep = (step: ApprovalStep) => {
  return !!step.nodes[0]?.externalNodeId;
};

const iconClass = (step: WrappedReviewStep) => {
  const { status } = step;
  return [
    status === "APPROVED" && "bg-success",
    status === "REJECTED" && "bg-warning",
    status === "CURRENT" && "bg-white border-[2px] border-info text-accent",
    status === "PENDING" && "bg-white border-[3px] border-gray-300",
  ];
};

const itemClass = (step: WrappedReviewStep) => {
  const { status } = step;
  return [
    status === "APPROVED" && "text-control-light",
    status === "REJECTED" && "text-control-light",
    status === "CURRENT" && "text-accent",
    status === "PENDING" && "text-control-placeholder",
  ];
};
</script>

<style>
.bb-approval-timeline
  .n-timeline-item
  .n-timeline-item-timeline
  .n-timeline-item-timeline__line {
  --line-padding: 4px;
  top: calc(var(--n-icon-size) + var(--line-padding)) !important;
  bottom: calc(var(--line-padding)) !important;
}
</style>
