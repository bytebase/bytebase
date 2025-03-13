<template>
  <NTimeline :icon-size="24">
    <NPopover
      v-for="step in steps"
      :disabled="step.status === 'APPROVED'"
      :key="step.index"
      placement="left"
    >
      <template #trigger>
        <NTimelineItem>
          <template #icon>
            <TimelineIcon :step="step" />
          </template>
          <template #default>
            <div class="flex flex-row gap-x-1">
              <div class="flex-1 truncate">
                <NPerformantEllipsis>
                  {{ approvalNodeText(step.step.nodes[0]) }}
                </NPerformantEllipsis>
              </div>
            </div>
          </template>
          <template #footer>
            <i18n-t
              keypath="custom-approval.issue-review.approved-by-n"
              tag="div"
              class="break-words break-all"
              v-if="step.status === 'APPROVED'"
            >
              <template #approver>
                <Approver
                  v-if="step.status === 'APPROVED'"
                  :step="step"
                  class="inline"
                >
                  <template
                    #title="{ approver }: { approver: User | undefined }"
                  >
                    <span>{{ approver?.title }}</span>
                  </template>
                </Approver>
              </template>
            </i18n-t>
          </template>
        </NTimelineItem>
      </template>

      <template #default>
        <div class="flex flex-col gap-y-2 text-sm max-w-[16rem] truncate">
          <ul class="w-full list-disc list-inside text-sm">
            <i18n-t
              keypath="custom-approval.issue-review.any-n-role-can-approve"
              tag="li"
              class="whitespace-pre-wrap"
            >
              <template #role>
                <span class="textlabel">
                  {{ approvalNodeText(step.step.nodes[0]) }}
                </span>
              </template>
            </i18n-t>
            <li
              v-if="!issue.projectEntity.allowSelfApproval"
              class="whitespace-pre-wrap"
            >
              {{
                $t(
                  "custom-approval.issue-review.issue-creators-cannot-approve-their-own-issue"
                )
              }}
            </li>
          </ul>
          <hr />
          <div
            v-if="step.candidates.length === 0"
            class="w-[14rem] text-wrap text-warning italic"
          >
            {{ $t("custom-approval.issue-review.no-one-matched") }}
          </div>
          <div
            v-else
            class="min-w-[8rem] max-w-[12rem] max-h-[18rem] flex flex-col text-control-light overflow-y-hidden"
          >
            <div class="flex-1 overflow-auto text-sm">
              <Candidate
                v-for="candidate in step.candidates"
                :key="candidate"
                :candidate="candidate"
              />
            </div>
          </div>
        </div>
      </template>
    </NPopover>
  </NTimeline>
</template>

<script lang="ts" setup>
import {
  NTimeline,
  NTimelineItem,
  NPopover,
  NPerformantEllipsis,
} from "naive-ui";
import { useIssueContext } from "@/components/IssueV1/logic";
import type { WrappedReviewStep } from "@/types";
import { type User } from "@/types/proto/v1/user_service";
import { approvalNodeText } from "@/utils";
import Approver from "./Approver.vue";
import Candidate from "./Candidate.vue";
import TimelineIcon from "./TimelineIcon.vue";

defineProps<{
  steps: WrappedReviewStep[];
}>();

const { issue } = useIssueContext();
</script>
