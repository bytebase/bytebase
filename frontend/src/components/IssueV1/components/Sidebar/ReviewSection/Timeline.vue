<template>
  <NTimeline :icon-size="28">
    <NTimelineItem v-for="step in steps" :key="step.index">
      <template #icon>
        <NPopover placement="left">
          <template #trigger>
            <TimelineIcon :step="step" />
          </template>

          <template #default>
            <div class="flex flex-col gap-y-2 text-sm max-w-[16rem] truncate">
              <i18n-t
                keypath="custom-approval.issue-review.any-n-role-can-approve"
                tag="div"
              >
                <template #role>
                  <span class="textlabel">
                    {{ approvalNodeText(step.step.nodes[0]) }}
                  </span>
                </template>
              </i18n-t>
              <hr />
              <template v-if="!isExternalApprovalStep(step.step)">
                <Approver
                  v-if="step.status === 'APPROVED'"
                  :step="step"
                  class="inline-flex flex-nowrap items-center whitespace-nowrap"
                />
                <Candidates v-else :step="step" />
              </template>
              <template v-else>
                <ExternalApprovalSyncButton />
              </template>
            </div>
          </template>
        </NPopover>
      </template>
      <template #default>
        <div class="flex flex-row gap-x-1">
          <div class="flex-1 truncate">
            <NPerformantEllipsis>
              {{ approvalNodeText(step.step.nodes[0]) }}
            </NPerformantEllipsis>
          </div>
          <div v-if="isExternalApprovalStep(step.step)" class="shrink-0">
            <ExternalApprovalSyncButton />
          </div>
        </div>
      </template>
      <template
        v-if="!isExternalApprovalStep(step.step) && step.status === 'APPROVED'"
        #footer
      >
        <i18n-t
          keypath="custom-approval.issue-review.approved-by-n"
          tag="div"
          class="break-words break-all"
        >
          <template #approver>
            <Approver
              v-if="step.status === 'APPROVED'"
              :step="step"
              class="inline"
            >
              <template #title="{ approver }: { approver: User | undefined }">
                <span>{{ approver?.title }}</span>
              </template>
            </Approver>
          </template>
        </i18n-t>
      </template>
    </NTimelineItem>
  </NTimeline>
</template>

<script lang="ts" setup>
import {
  NTimeline,
  NTimelineItem,
  NPopover,
  NPerformantEllipsis,
} from "naive-ui";
import { WrappedReviewStep } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
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
</script>
