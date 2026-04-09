<template>
  <div class="w-full flex flex-col p-4 gap-4">
    <!-- 1. Checks - Health status -->
    <div v-if="isDatabaseChangePlan" class="flex flex-col gap-2">
      <NAlert v-if="showChecksManualRolloutHint" type="warning">
        {{ $t("issue.checks-manual-rollout-hint") }}
      </NAlert>
      <ChecksSection />
    </div>

    <!-- 2. Approval Flow - Reviewers -->
    <ApprovalFlowSection :issue="issue" />

    <!-- 3. Labels - Metadata -->
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
import { isApprovalCompleted } from "@/components/Plan/logic/approval";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { usePlanCheckStatus } from "@/components/Plan/logic/usePlanCheckStatus";
import { issueServiceClientConnect } from "@/connect";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import {
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContextWithIssue } from "../../../logic/context";
import ApprovalFlowSection from "./ApprovalFlowSection/ApprovalFlowSection.vue";
import ChecksSection from "./ChecksSection.vue";

const { t } = useI18n();
const { plan, issue } = usePlanContextWithIssue();
const { project } = useCurrentProjectV1();
const { refreshResources } = useResourcePoller();
const { getOverallStatus: planCheckStatus } = usePlanCheckStatus(plan);

const isDatabaseChangePlan = computed(
  () =>
    plan.value.specs.length > 0 &&
    plan.value.specs.every(
      (spec) => spec.config?.case === "changeDatabaseConfig"
    )
);

const issueApprovalCompleted = computed(() => isApprovalCompleted(issue.value));

const showChecksManualRolloutHint = computed(() => {
  return (
    !plan.value.hasRollout &&
    planCheckStatus.value === Advice_Level.ERROR &&
    issueApprovalCompleted.value
  );
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
