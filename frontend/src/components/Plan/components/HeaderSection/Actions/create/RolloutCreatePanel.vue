<template>
  <CommonDrawer
    :show="show"
    :title="$t('issue.create-rollout')"
    :loading="loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 h-full px-1">
        <!-- Warning alert with messages -->
        <NAlert
          v-if="warningMessages.length > 0"
          type="warning"
          :title="$t('common.notices')"
        >
          <ul class="list-disc list-inside flex flex-col gap-y-1">
            <li
              v-for="(warning, index) in warningMessages"
              :key="index"
              class="text-sm"
            >
              {{ warning }}
            </li>
          </ul>
        </NAlert>

        <!-- Approval Flow Section -->
        <ApprovalFlowSection v-if="issue" :issue="issue" />

        <!-- Plan Check Status -->
        <div class="w-full flex flex-col gap-2">
          <span class="font-medium text-control shrink-0">{{
            $t("plan.navigator.checks")
          }}</span>
          <PlanCheckStatusCount :plan="plan" />
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex flex-row justify-between items-center gap-x-2">
        <!-- Bypass checkbox -->
        <div v-if="hasWarnings" class="flex items-center">
          <NCheckbox v-model:checked="bypassWarnings" :disabled="loading">
            {{ $t("rollout.bypass-stage-requirements") }}
          </NCheckbox>
        </div>
        <div v-else />

        <div class="flex justify-end gap-x-2">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>

          <NButton
            :disabled="hasWarnings && !bypassWarnings"
            type="primary"
            :loading="loading"
            @click="handleConfirm"
          >
            {{ $t("issue.create-rollout") }}
          </NButton>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NAlert, NButton, NCheckbox } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ApprovalFlowSection } from "@/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection";
import PlanCheckStatusCount from "@/components/Plan/components/PlanCheckStatusCount.vue";
import { usePlanContext } from "@/components/Plan/logic";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { pushNotification } from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
} from "@/utils";
import type { ActionContext } from "../registry/types";

const props = defineProps<{
  show: boolean;
  context: ActionContext;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "confirm"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const { events } = usePlanContext();

// Use context from props for consistency
const plan = computed(() => props.context.plan);
const issue = computed(() => props.context.issue);
const project = computed(() => props.context.project);
const warnings = computed(() => props.context.rolloutCreationWarnings);

const loading = ref(false);
const bypassWarnings = ref(false);

// Build warning messages based on current state
const warningMessages = computed(() => {
  const messages: string[] = [];

  if (warnings.value.approvalNotReady) {
    messages.push(
      t("project.settings.issue-related.require-issue-approval.description")
    );
  }

  if (warnings.value.planChecksRunning) {
    messages.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      )
    );
  } else if (warnings.value.planChecksFailed) {
    messages.push(
      t(
        "project.settings.issue-related.require-plan-check-no-error.description"
      )
    );
  }

  return messages;
});

const hasWarnings = computed(() => warnings.value.hasAny);

const resetState = () => {
  bypassWarnings.value = false;
};

const handleConfirm = async () => {
  if (loading.value) return;

  loading.value = true;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: plan.value.name,
    });
    const createdRollout =
      await rolloutServiceClientConnect.createRollout(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });

    events.emit("status-changed", { eager: true });
    emit("confirm");

    router.push({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        planId: extractPlanUIDFromRolloutName(createdRollout.name),
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: String(error),
    });
  } finally {
    loading.value = false;
    emit("close");
  }
};
</script>
