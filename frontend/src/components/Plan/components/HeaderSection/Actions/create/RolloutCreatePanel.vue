<template>
  <CommonDrawer
    :show="show"
    :title="$t('issue.create-rollout')"
    :loading="loading"
    @show="bypassWarnings = false"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 h-full px-1">
        <!-- Error alert (blocking) -->
        <NAlert v-if="errorMessages.length > 0" type="error" :title="$t('common.error')">
          <ul class="list-disc list-inside text-sm">
            <li v-for="msg in errorMessages" :key="msg">{{ msg }}</li>
          </ul>
        </NAlert>

        <!-- Warning alert (bypassable) -->
        <NAlert
          v-else-if="warningMessages.length > 0"
          type="warning"
          :title="$t('common.notices')"
        >
          <ul class="list-disc list-inside text-sm">
            <li v-for="msg in warningMessages" :key="msg">{{ msg }}</li>
          </ul>
        </NAlert>

        <ApprovalFlowSection v-if="issue" :issue="issue" />

        <div class="w-full flex flex-col gap-2">
          <span class="font-medium text-control">{{
            $t("plan.navigator.checks")
          }}</span>
          <PlanCheckStatusCount :plan="plan" />
          <span v-if="!hasAnyChecks" class="text-sm text-control-placeholder">
            {{ $t("plan.overview.no-checks") }}
          </span>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex justify-between items-center gap-x-2">
        <NCheckbox
          v-if="warningMessages.length > 0 && errorMessages.length === 0"
          v-model:checked="bypassWarnings"
          :disabled="loading"
        >
          {{ $t("rollout.bypass-stage-requirements") }}
        </NCheckbox>
        <div v-else />

        <div class="flex gap-x-2">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="errorMessages.length > 0 || (warningMessages.length > 0 && !bypassWarnings)"
            :loading="loading"
            @click="handleConfirm"
          >
            {{ $t("common.confirm") }}
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
import { usePlanCheckStatus, usePlanContext } from "@/components/Plan/logic";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { pushNotification } from "@/store";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
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

const plan = computed(() => props.context.plan);
const issue = computed(() => props.context.issue);
const project = computed(() => props.context.project);
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

const loading = ref(false);
const bypassWarnings = ref(false);

// Errors: require_*=true and condition not met (blocking)
const errorMessages = computed(() => {
  const msgs: string[] = [];
  if (project.value.requireIssueApproval && !props.context.issueApproved) {
    msgs.push(
      t("project.settings.issue-related.require-issue-approval.description")
    );
  }
  if (
    project.value.requirePlanCheckNoError &&
    props.context.validation.planChecksFailed
  ) {
    msgs.push(
      t(
        "project.settings.issue-related.require-plan-check-no-error.description"
      )
    );
  }
  return msgs;
});

// Warnings: require_*=false and condition not met (bypassable)
const warningMessages = computed(() => {
  const msgs: string[] = [];
  if (!project.value.requireIssueApproval && !props.context.issueApproved) {
    msgs.push(
      t("project.settings.issue-related.require-issue-approval.description")
    );
  }
  if (props.context.validation.planChecksRunning) {
    msgs.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      )
    );
  } else if (
    !project.value.requirePlanCheckNoError &&
    props.context.validation.planChecksFailed
  ) {
    msgs.push(
      t(
        "project.settings.issue-related.require-plan-check-no-error.description"
      )
    );
  }
  return msgs;
});

const handleConfirm = async () => {
  if (loading.value) return;
  loading.value = true;

  try {
    const createdRollout = await rolloutServiceClientConnect.createRollout(
      create(CreateRolloutRequestSchema, { parent: plan.value.name })
    );

    // Mark issue as done after rollout created.
    if (issue.value) {
      await issueServiceClientConnect.batchUpdateIssuesStatus(
        create(BatchUpdateIssuesStatusRequestSchema, {
          parent: project.value.name,
          issues: [issue.value.name],
          status: IssueStatus.DONE,
        })
      );
    }

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
