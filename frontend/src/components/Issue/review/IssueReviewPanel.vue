<template>
  <div v-if="ready && wrappedSteps" class="mt-1">
    <div
      v-for="step in wrappedSteps"
      :key="step.index"
      class="flex items-start gap-x-2 relative"
    >
      <div
        class="w-5 h-5 rounded-full flex items-center justify-center text-xs shrink-0"
        :class="step.iconClass"
      >
        <heroicons-outline:thumb-up
          v-if="step.status === 'DONE'"
          class="w-3.5 h-3.5 text-white"
        />
        <div
          v-else-if="step.status === 'CURRENT'"
          class="w-1.5 h-1.5 rounded-full bg-info"
        ></div>
        <template v-else>
          {{ step.index + 1 }}
        </template>
      </div>

      <div
        class="flex-1 flex text-sm overflow-hidden"
        :class="[
          step.index < wrappedSteps.length - 1 && 'min-h-[2.5rem]',
          step.status === 'DONE' && 'text-control-light',
          step.status === 'CURRENT' && 'text-accent',
          step.status === 'PENDING' && 'text-control-placeholder',
        ]"
      >
        <div class="whitespace-nowrap shrink-0">
          {{ approvalNodeGroupValueText(step.step.nodes[0].groupValue!) }}
        </div>
        <div class="mr-1.5 shrink-0">:</div>
        <div class="flex-1 overflow-hidden">
          <NEllipsis
            v-if="step.status === 'DONE'"
            class="inline-block"
            :class="step.approver?.name === currentUserName && 'font-bold'"
          >
            <span>{{ step.approver?.title }}</span>
            <span v-if="step.approver?.name === currentUserName">
              ({{ $t("custom-approval.issue-review.you") }})
            </span>
          </NEllipsis>
          <!-- <Candidates v-else :candidates="step.candidates" /> -->
          <NEllipsis
            v-else
            class="flex-1 truncate"
            :tooltip="{
              raw: true,
              showArrow: false,
            }"
          >
            <div
              v-for="(user, i) in step.candidates"
              :key="user.name"
              class="inline-flex flex-nowrap truncate"
            >
              <span
                :class="user.name === currentUserName && 'font-bold'"
                class="truncate"
              >
                {{ user.title }}
              </span>
              <span v-if="user.name === currentUserName" class="font-bold ml-1">
                ({{ $t("custom-approval.issue-review.you") }})
              </span>
              <span v-if="i < step.candidates.length - 1" class="mr-1">,</span>
            </div>

            <template #tooltip>
              <div
                class="w-[12rem] max-h-[18rem] bg-white text-control-light py-1 px-2 overflow-auto divide-y"
              >
                <div
                  v-for="user in step.candidates"
                  :key="user.name"
                  class="py-1"
                  :class="[user.name === currentUserName && 'font-bold']"
                >
                  <span class="whitespace-nowrap">{{ user.title }}</span>
                  <span v-if="user.name === currentUserName" class="ml-1">
                    ({{ $t("custom-approval.issue-review.you") }})
                  </span>
                </div>
              </div>
            </template>
          </NEllipsis>
        </div>
      </div>

      <div
        v-if="step.index < wrappedSteps.length - 1"
        class="absolute w-5 h-[calc(100%-8px-1.25rem)] bottom-[4px] flex items-center justify-center shrink-0"
      >
        <div class="w-0.5 h-full bg-block-border"></div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NEllipsis } from "naive-ui";

import {
  candidatesOfApprovalStep,
  extractUserEmail,
  useAuthStore,
  useUserStore,
} from "@/store";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { approvalNodeGroupValueText, VueClass } from "@/utils";
import { useIssueLogic } from "../logic";
import { Issue } from "@/types";
import { ApprovalStep } from "@/types/proto/v1/review_service";
import { User } from "@/types/proto/v1/auth_service";

const userStore = useUserStore();
const issueLogic = useIssueLogic();
const currentUserName = computed(() => useAuthStore().currentUser.name);
const issue = computed(() => issueLogic.issue.value as Issue);
const context = useIssueReviewContext();
const { ready, flow, done } = context;

type WrappedStep = {
  index: number;
  iconClass: VueClass;
  step: ApprovalStep;
  status: "DONE" | "CURRENT" | "PENDING";
  approver: User | undefined;
  candidates: User[];
};

const wrappedSteps = computed(() => {
  const steps = flow.value.template.flow?.steps;
  const currentStepIndex = flow.value.currentStepIndex ?? -1;

  const statusOfStep = (index: number) => {
    if (done.value) return "DONE";
    if (index < currentStepIndex) return "DONE";
    if (index === currentStepIndex) return "CURRENT";
    return "PENDING";
  };
  const approverOfStep = (index: number) => {
    const principal = flow.value.approvers[index]?.principal;
    if (!principal) return undefined;
    const email = extractUserEmail(principal);
    return userStore.getUserByEmail(email);
  };
  const candidatesOfStep = (index: number) => {
    const step = steps?.[index];
    if (!step) return [];
    const users = candidatesOfApprovalStep(issue.value, step);
    const idx = users.indexOf(currentUserName.value);
    if (idx > 0) {
      users.splice(idx, 1);
      users.unshift(currentUserName.value);
    }
    return users.map((user) => userStore.getUserByName(user)!);
  };
  const classOfStep = (index: number) => {
    const status = statusOfStep(index);
    if (status === "DONE") {
      return "bg-success";
    }
    if (status === "CURRENT") {
      return "bg-white border-[2px] border-info text-accent";
    }
    return "bg-white border-[3px] border-gray-300";
  };

  return steps?.map<WrappedStep>((step, index) => ({
    index,
    step,
    iconClass: classOfStep(index),
    status: statusOfStep(index),
    approver: approverOfStep(index),
    candidates: candidatesOfStep(index),
  }));
});
</script>
