<template>
  <div class="w-full shrink-0 flex flex-col text-sm">
    <!-- Plan group: Created by + Refresh + Status + Checks -->
    <div class="flex flex-col gap-3 pb-3">
      <div v-if="!isCreating" class="flex flex-col">
        <p class="text-xs text-control-placeholder">
          {{ $t("plan.sidebar.created-by-at", { user: creatorEmail, time: createdTimeDisplay }) }}
        </p>
      </div>
      <div>
        <h4 class="textinfolabel mb-1">{{ $t("common.status") }}</h4>
        <div class="flex items-center gap-2">
          <div class="w-2.5 h-2.5 rounded-full" :class="statusInfo.dotClass" />
          <span class="text-sm font-medium text-main">{{ statusInfo.label }}</span>
        </div>
      </div>
      <div v-if="hasAnyChecks">
        <h4 class="textinfolabel mb-1">{{ $t("plan.navigator.checks") }}</h4>
        <PlanCheckStatusCount :plan="plan" />
      </div>
    </div>

    <!-- Issue group: Approval flow + link + Labels -->
    <div v-if="issue" class="flex flex-col gap-3 py-3 border-t">
      <ApprovalFlowSection :issue="issue" />
      <IssueLabels
        :project="project"
        :value="issue.labels || []"
        :disabled="!allowChangeLabels"
        @update:value="onIssueLabelsUpdate"
      />
    </div>

    <!-- Rollout group: Stages -->
    <div v-if="rollout && rollout.stages.length > 0" class="flex flex-col gap-3 py-3 border-t">
      <h4 class="textinfolabel">{{ $t("rollout.stage.self", 2) }}</h4>
      <div
        v-for="stage in rollout.stages"
        :key="stage.name"
        class="flex items-center gap-2"
      >
        <TaskStatus :status="getStageStatus(stage)" size="small" disabled />
        <EnvironmentV1Name
          :environment="environmentStore.getEnvironmentByName(stage.environment)"
          :link="false"
        />
        <span class="text-control-placeholder">
          ({{ completedTaskCount(stage) }}/{{ stage.tasks.length }})
        </span>
      </div>
    </div>

    <!-- Placeholder when no issue yet -->
    <div v-if="!issue" class="pt-3 border-t">
      <p class="text-sm text-control-placeholder">
        {{ $t("plan.sidebar.future-sections-hint") }}
      </p>
    </div>

    <div v-if="!isCreating" class="pt-1">
      <RefreshIndicator />
    </div>
  </div>
</template>

<script setup lang="ts">
import { clone, create } from "@bufbuild/protobuf";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IssueLabels from "@/components/IssueV1/components/Sidebar/IssueLabels.vue";
import TaskStatus from "@/components/RolloutV1/components/Task/TaskStatus.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { issueServiceClientConnect } from "@/connect";
import {
  extractUserEmail,
  pushNotification,
  useCurrentProjectV1,
  useEnvironmentV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  IssueSchema,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  getRolloutStatus,
  getStageStatus,
  hasProjectPermissionV2,
} from "@/utils";
import { humanizeTs } from "@/utils/util";
import { usePlanCheckStatus, usePlanContext } from "../../logic";
import { useResourcePoller } from "../../logic/poller";
import ApprovalFlowSection from "../IssueReviewView/Sidebar/ApprovalFlowSection/ApprovalFlowSection.vue";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";
import RefreshIndicator from "../RefreshIndicator.vue";

const { t } = useI18n();
const { isCreating, plan, issue, rollout } = usePlanContext();
const { project } = useCurrentProjectV1();
const { refreshResources } = useResourcePoller();
const environmentStore = useEnvironmentV1Store();
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

// --- Status ---

type StatusInfo = { label: string; dotClass: string };

const getReviewStatusInfo = (): StatusInfo => {
  if (!issue.value) {
    return { label: t("common.draft"), dotClass: "bg-control-placeholder" };
  }

  if (issue.value.status === IssueStatus.DONE) {
    return { label: t("common.approved"), dotClass: "bg-success" };
  }

  switch (issue.value.approvalStatus) {
    case Issue_ApprovalStatus.APPROVED:
    case Issue_ApprovalStatus.SKIPPED:
      return { label: t("common.approved"), dotClass: "bg-success" };
    case Issue_ApprovalStatus.REJECTED:
      return { label: t("common.rejected"), dotClass: "bg-warning" };
    default:
      return { label: t("common.in-review"), dotClass: "bg-accent" };
  }
};

const getRolloutStatusInfo = (): StatusInfo => {
  if (!rollout.value) {
    return getReviewStatusInfo();
  }

  switch (getRolloutStatus(rollout.value)) {
    case Task_Status.DONE:
    case Task_Status.SKIPPED:
      return { label: t("common.done"), dotClass: "bg-success" };
    case Task_Status.FAILED:
      return { label: t("common.failed"), dotClass: "bg-error" };
    case Task_Status.RUNNING:
    case Task_Status.PENDING:
      return { label: t("common.deploying"), dotClass: "bg-accent" };
    case Task_Status.CANCELED:
      return {
        label: t("common.canceled"),
        dotClass: "bg-control-placeholder",
      };
    case Task_Status.NOT_STARTED:
    default:
      return {
        label: t("common.not-started"),
        dotClass: "bg-control-placeholder",
      };
  }
};

const statusInfo = computed<StatusInfo>(() => {
  if (isCreating.value) {
    return { label: t("common.creating"), dotClass: "bg-control-placeholder" };
  }
  if (plan.value.state === State.DELETED) {
    return { label: t("common.closed"), dotClass: "bg-control-placeholder" };
  }
  if (issue.value?.status === IssueStatus.CANCELED) {
    return { label: t("common.closed"), dotClass: "bg-control-placeholder" };
  }
  if (rollout.value) {
    return getRolloutStatusInfo();
  }
  return getReviewStatusInfo();
});

// --- Creator / Created ---

const creatorEmail = computed(() => extractUserEmail(plan.value.creator));

const createdTimeDisplay = computed(() => {
  const ts = getTimeForPbTimestampProtoEs(plan.value.createTime, 0);
  if (!ts) return "";
  return humanizeTs(ts / 1000);
});

// --- Labels ---

const allowChangeLabels = computed(() => {
  if (!issue.value || issue.value.status !== IssueStatus.OPEN) return false;
  return hasProjectPermissionV2(project.value, "bb.issues.update");
});

const onIssueLabelsUpdate = async (labels: string[]) => {
  if (!issue.value) return;
  try {
    const issuePatch = clone(IssueSchema, issue.value);
    issuePatch.labels = labels;
    const request = create(UpdateIssueRequestSchema, {
      issue: issuePatch,
      updateMask: { paths: ["labels"] },
    });
    await issueServiceClientConnect.updateIssue(request);
    refreshResources(["issue"], true);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch {
    // Ignore — label update is non-critical
  }
};

// --- Stages ---

const completedTaskCount = (stage: Stage) => {
  return stage.tasks.filter(
    (task) =>
      task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
  ).length;
};
</script>
