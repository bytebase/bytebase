<template>
  <NTimeline
    :horizontal="true"
    :icon-size="28"
    class="bb-approval-timeline-horizontal"
  >
    <NTimelineItem v-for="step in steps" :key="step.index">
      <template #icon>
        <NPopover placement="bottom">
          <template #trigger>
            <TimelineIcon :step="step" />
          </template>

          <template #default>
            <div class="flex flex-col gap-y-2 text-sm">
              <div class="whitespace-nowrap textlabel">
                {{ approvalNodeText(step.step.nodes[0]) }}
              </div>
              <hr />
              <template v-if="!isExternalApprovalStep(step.step)">
                <Approver v-if="step.status === 'APPROVED'" :step="step" />
                <Candidates v-else :step="step" />
              </template>
              <template v-else>
                <ExternalApprovalSyncButton />
              </template>
            </div>
          </template>
        </NPopover>
      </template>

      <template v-if="false" #default>
        <div class="flex text-sm whitespace-nowrap" :class="itemClass(step)">
          {{ approvalNodeText(step.step.nodes[0]) }}
        </div>
      </template>
    </NTimelineItem>
  </NTimeline>
</template>

<script lang="ts" setup>
import { NTimeline, NTimelineItem, NPopover } from "naive-ui";
import { WrappedReviewStep } from "@/types";
import { ApprovalStep } from "@/types/proto/v1/issue_service";
import { approvalNodeText } from "@/utils";
import Approver from "./Approver.vue";
import Candidates from "./Candidates.vue";
import ExternalApprovalSyncButton from "./ExternalApprovalNodeSyncButton.vue";
import TimelineIcon from "./TimelineIcon.vue";

defineProps<{
  steps: WrappedReviewStep[];
}>();

const isExternalApprovalStep = (step: ApprovalStep) => {
  return !!step.nodes[0]?.externalNodeId;
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
.bb-approval-timeline-horizontal .n-timeline-item {
  padding-right: 20px !important;
}
.bb-approval-timeline-horizontal .n-timeline-item:last-child {
  padding-right: 0 !important;
}
.bb-approval-timeline-horizontal .n-timeline-item .n-timeline-item-timeline {
  position: static;
  height: calc(var(--n-icon-size)) !important;
}
.bb-approval-timeline-horizontal
  .n-timeline-item
  .n-timeline-item-timeline
  .n-timeline-item-timeline__line {
  --line-padding: 4px;
  left: calc(var(--n-icon-size) + var(--line-padding)) !important;
  right: calc(var(--line-padding)) !important;
  height: 3px !important;
}
.bb-approval-timeline-horizontal .n-timeline-item .n-timeline-item-content {
  margin-top: 2px !important;
}
.bb-approval-timeline-horizontal
  .n-timeline-item
  .n-timeline-item-content
  .n-timeline-item-content__meta {
  display: none;
}
</style>
