<template>
  <div class="mt-1.5">
    <p class="text-sm text-control-placeholder">
      {{ $t("plan.phase.deploy-description") }}
    </p>

    <ul class="mt-2.5 max-w-[28rem] space-y-1">
      <li
        v-for="item in requirementItems"
        :key="item.key"
        class="flex items-start gap-2 py-1"
      >
        <div class="flex items-start gap-2 min-w-0">
          <component
            :is="item.icon"
            class="mt-0.5 h-3.5 w-3.5 shrink-0"
            :class="item.iconClass"
          />
          <div class="min-w-0">
            <div class="flex items-center gap-1.5 flex-wrap">
              <span class="text-xs font-medium text-control">
                {{ item.label }}
              </span>
              <span
                v-if="item.required"
                class="text-[10px] font-medium text-error/80"
              >
                *
              </span>
              <span
                class="text-[11px] font-medium"
                :class="item.statusClass"
              >
                {{ item.tagLabel }}
              </span>
            </div>
            <p class="mt-0.5 text-[11px] text-control-placeholder">
              {{ item.description }}
            </p>
          </div>
        </div>
      </li>
    </ul>

    <div
      v-if="showManualCreateRolloutHint"
      class="mt-3 flex max-w-[28rem] flex-col items-start gap-y-2"
    >
      <p class="text-xs text-control-placeholder">
        {{ manualCreateRolloutDescription }}
      </p>
      <NButton
        v-if="canCreateRollout"
        secondary
        strong
        size="small"
        @click="showRolloutCreatePanel = true"
      >
        {{ $t("plan.phase.create-rollout-action") }}
      </NButton>
    </div>

    <RolloutCreatePanel
      :show="showRolloutCreatePanel"
      :context="actionContext"
      @close="showRolloutCreatePanel = false"
      @confirm="showRolloutCreatePanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import {
  CheckCircle2Icon,
  CircleAlertIcon,
  Clock3Icon,
  MinusCircleIcon,
} from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentProjectV1 } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanContext } from "../../logic";
import RolloutCreatePanel from "../HeaderSection/Actions/create/RolloutCreatePanel.vue";
import { useActionRegistry } from "../HeaderSection/Actions/registry/useActionRegistry";

const { plan, issue } = usePlanContext();
const { context: actionContext } = useActionRegistry();
const { project } = useCurrentProjectV1();
const { t } = useI18n();

const showRolloutCreatePanel = ref(false);

const canCreateRollout = computed(() => {
  if (!issue.value) return false;
  if (plan.value.hasRollout) return false;
  if (plan.value.state !== State.ACTIVE) return false;
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!actionContext.value.permissions.createRollout) return false;
  return true;
});

const showManualCreateRolloutHint = computed(() => {
  if (!issue.value) return false;
  if (plan.value.hasRollout) return false;
  if (plan.value.state !== State.ACTIVE) return false;
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (
    project.value.requireIssueApproval &&
    !actionContext.value.issueApproved
  ) {
    return false;
  }
  if (
    project.value.requirePlanCheckNoError &&
    actionContext.value.validation.planChecksFailed
  ) {
    return false;
  }
  return (
    !project.value.requireIssueApproval ||
    !project.value.requirePlanCheckNoError
  );
});

const manualCreateRolloutDescription = computed(() => {
  if (actionContext.value.permissions.createRollout) {
    return t("plan.phase.deploy-manual-create-description");
  }
  return t("plan.phase.deploy-manual-create-description-readonly");
});

const requirementItems = computed(() => {
  const approvalItem = project.value.requireIssueApproval
    ? actionContext.value.issueApproved
      ? {
          key: "approval",
          label: t("plan.phase.deploy-approval-must-complete"),
          description: t("plan.phase.deploy-approval-ready"),
          tagLabel: t("common.done"),
          statusClass: "text-success",
          icon: CheckCircle2Icon,
          iconClass: "text-success/80",
          required: true,
        }
      : {
          key: "approval",
          label: t("plan.phase.deploy-approval-must-complete"),
          description: t("plan.phase.deploy-approval-pending"),
          tagLabel: t("common.pending"),
          statusClass: "text-warning",
          icon: Clock3Icon,
          iconClass: "text-warning/80",
          required: true,
        }
    : {
        key: "approval",
        label: t("plan.phase.deploy-approval-must-complete"),
        description: t("plan.phase.deploy-approval-optional"),
        tagLabel: t("common.optional"),
        statusClass: "text-control-placeholder",
        icon: MinusCircleIcon,
        iconClass: "text-control-placeholder",
        required: false,
      };

  const checksItem = project.value.requirePlanCheckNoError
    ? actionContext.value.validation.planChecksRunning
      ? {
          key: "checks",
          label: t("plan.phase.deploy-checks-must-pass"),
          description: t("plan.phase.deploy-checks-running"),
          tagLabel: t("common.in-progress"),
          statusClass: "text-warning",
          icon: Clock3Icon,
          iconClass: "text-warning/80",
          required: true,
        }
      : actionContext.value.validation.planChecksFailed
        ? {
            key: "checks",
            label: t("plan.phase.deploy-checks-must-pass"),
            description: t("plan.phase.deploy-checks-blocked"),
            tagLabel: t("common.failed"),
            statusClass: "text-error",
            icon: CircleAlertIcon,
            iconClass: "text-error/80",
            required: true,
          }
        : {
            key: "checks",
            label: t("plan.phase.deploy-checks-must-pass"),
            description: t("plan.phase.deploy-checks-ready"),
            tagLabel: t("common.done"),
            statusClass: "text-success",
            icon: CheckCircle2Icon,
            iconClass: "text-success/80",
            required: true,
          }
    : {
        key: "checks",
        label: t("plan.phase.deploy-checks-must-pass"),
        description: t("plan.phase.deploy-checks-optional"),
        tagLabel: t("common.optional"),
        statusClass: "text-control-placeholder",
        icon: MinusCircleIcon,
        iconClass: "text-control-placeholder",
        required: false,
      };

  return [checksItem, approvalItem];
});
</script>
