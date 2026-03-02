<template>
  <NTimelineItem :type="timelineItemType">
    <template #icon>
      <div
        class="p-1 rounded-full flex items-center justify-center z-10"
        :class="iconClass"
      >
        <ThumbsUp v-if="status === 'approved'" class="w-4 h-4 text-white" />
        <X v-else-if="status === 'rejected'" class="w-4 h-4 text-white" />
        <User v-else-if="status === 'current'" class="w-4 h-4 text-white" />
        <div v-else class="flex w-4 h-4 justify-center items-center">
          <span class="text-sm font-medium text-gray-600">{{ stepNumber }}</span>
        </div>
      </div>
    </template>

    <div>
      <div class="text-sm font-medium text-gray-900">
        {{ roleName }}
      </div>

      <div class="mt-1 text-sm text-gray-600">
        <!-- Approved -->
        <div v-if="status === 'approved'" class="flex flex-row items-center gap-1">
          <span class="text-xs">
            {{ $t("custom-approval.issue-review.approved-by") }}
          </span>
          <ApprovalUserView
            v-if="stepApprover"
            :candidate="stepApprover.principal"
          />
        </div>

        <!-- Rejected -->
        <div v-else-if="status === 'rejected'" class="flex flex-col gap-1">
          <div class="flex flex-row items-center gap-1">
            <span class="text-xs">
              {{ $t("custom-approval.issue-review.rejected-by") }}
            </span>
            <ApprovalUserView
              v-if="stepApprover"
              :candidate="stepApprover.principal"
            />
          </div>
          <div v-if="canReRequest && !readonly" class="mt-1">
            <NButton
              size="tiny"
              :loading="reRequesting"
              @click="handleReRequestReview"
            >
              <template #icon>
                <RotateCcwIcon class="w-3 h-3" />
              </template>
              {{ $t("custom-approval.issue-review.re-request-review") }}
            </NButton>
          </div>
        </div>

        <!-- Current -->
        <div v-else-if="status === 'current'" class="flex flex-col gap-1">
          <template v-if="!readonly">
            <PotentialApprovers :users="potentialApprovers" />
            <div
              v-if="showSelfApprovalTip"
              class="px-1 py-0.5 border rounded-sm text-xs bg-yellow-50 border-yellow-600 text-yellow-600"
            >
              {{ $t("custom-approval.issue-review.self-approval-not-allowed") }}
            </div>
          </template>
        </div>
      </div>
    </div>
  </NTimelineItem>
</template>

<script setup lang="ts">
import { RotateCcwIcon, ThumbsUp, User, X } from "lucide-vue-next";
import { NButton, NTimelineItem } from "naive-ui";
import { computed, toRef } from "vue";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import ApprovalUserView from "./ApprovalUserView.vue";
import PotentialApprovers from "./PotentialApprovers.vue";
import { useApprovalStep } from "./useApprovalStep";

const props = withDefaults(
  defineProps<{
    step: string;
    stepIndex: number;
    stepNumber: number;
    issue: Issue;
    readonly?: boolean;
  }>(),
  {
    readonly: false,
  }
);

const {
  status,
  stepApprover,
  roleName,
  canReRequest,
  reRequesting,
  handleReRequestReview,
  potentialApprovers,
  showSelfApprovalTip,
} = useApprovalStep(
  toRef(props, "issue"),
  toRef(props, "step"),
  toRef(props, "stepIndex")
);

const timelineItemType = computed(() => {
  switch (status.value) {
    case "approved":
      return "success";
    case "rejected":
      return "warning";
    case "current":
      return "info";
    default:
      return "default";
  }
});

const iconClass = computed(() => {
  switch (status.value) {
    case "approved":
      return "bg-green-500";
    case "rejected":
      return "bg-yellow-500";
    case "current":
      return "bg-blue-500";
    default:
      return "bg-gray-200";
  }
});
</script>
