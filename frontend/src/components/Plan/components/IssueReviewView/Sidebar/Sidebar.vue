<template>
  <div class="w-full flex flex-col p-4 gap-4">
    <!-- 1. Stages - Progress indicator -->
    <StagesSection v-if="isDatabaseChangePlan" />

    <!-- 2. Checks rollout hint -->
    <NAlert v-if="showChecksBlockRolloutHint" type="error">
      {{ $t("issue.checks-block-rollout-hint") }}
    </NAlert>
    <NAlert v-else-if="showForceRolloutHint" type="warning">
      {{ $t("issue.force-rollout-hint") }}
    </NAlert>

    <!-- 3. Checks - Health status -->
    <ChecksSection v-if="isDatabaseChangePlan" />

    <!-- 4. Approval Flow - Reviewers -->
    <ApprovalFlowSection :issue="issue" />

    <!-- 5. Labels - Metadata -->
    <IssueLabels
      :project="project"
      :value="issue.labels || []"
      :disabled="!allowChange"
      @update:value="onIssueLabelsUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NAlert } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IssueLabels from "@/components/IssueV1/components/Sidebar/IssueLabels.vue";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { issueServiceClientConnect } from "@/connect";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanCheckStatus } from "../../../logic";
import { usePlanContextWithIssue } from "../../../logic/context";
import ApprovalFlowSection from "./ApprovalFlowSection/ApprovalFlowSection.vue";
import ChecksSection from "./ChecksSection.vue";
import StagesSection from "./StagesSection.vue";

const { t } = useI18n();
const { plan, issue } = usePlanContextWithIssue();
const { project } = useCurrentProjectV1();
const { refreshResources } = useResourcePoller();
const { hasErrors: checksHaveErrors, hasRunning: checksRunning } =
  usePlanCheckStatus(plan);

const isDatabaseChangePlan = computed(() =>
  plan.value.specs.some((spec) => spec.config?.case === "changeDatabaseConfig")
);

const isApproved = computed(() => {
  const status = issue.value.approvalStatus;
  if (status === Issue_ApprovalStatus.CHECKING) return false;
  const roles = issue.value.approvalTemplate?.flow?.roles ?? [];
  return (
    status === Issue_ApprovalStatus.APPROVED ||
    status === Issue_ApprovalStatus.SKIPPED ||
    roles.length === 0
  );
});

const showChecksHint = computed(() => {
  return (
    issue.value.status === IssueStatus.OPEN &&
    plan.value.state === State.ACTIVE &&
    isDatabaseChangePlan.value &&
    isApproved.value &&
    checksHaveErrors.value &&
    !checksRunning.value &&
    !plan.value.hasRollout
  );
});

const showChecksBlockRolloutHint = computed(() => {
  return showChecksHint.value && project.value.requirePlanCheckNoError;
});

const showForceRolloutHint = computed(() => {
  return showChecksHint.value && !project.value.requirePlanCheckNoError;
});

const allowChange = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  return hasProjectPermissionV2(project.value, "bb.issues.update");
});

const onIssueLabelsUpdate = async (labels: string[]) => {
  const issuePatch = {
    ...issue.value,
    labels,
  };
  const request = create(UpdateIssueRequestSchema, {
    issue: issuePatch,
    updateMask: { paths: ["labels"] },
  });
  await issueServiceClientConnect.updateIssue(request);
  refreshResources(["issue"], true /** force */);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
