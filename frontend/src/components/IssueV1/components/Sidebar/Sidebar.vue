<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <ReleaseInfo />
    <TaskCheckSummarySection />

    <!-- Review section -->
    <ApprovalFlowSection
      v-if="!isCreating"
      :issue="issue"
      @issue-updated="
        events.emit('status-changed', {
          eager: true,
        })
      "
    />
    <div v-else class="flex flex-col gap-y-1">
      <div class="textlabel flex items-center gap-x-1">
        {{ $t("issue.approval-flow.self") }}
        <FeatureBadge :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      </div>
      <div class="text-control-placeholder text-xs">
        {{ $t("issue.approval-flow.pre-issue-created-tips") }}
      </div>
    </div>

    <IssueLabels
      :project="project"
      :value="issue.labels"
      :disabled="!allowChange"
      @update:value="onIssueLabelsUpdate"
    />

    <div
      v-show="
        selectedSpec && (shouldShowPreBackupSection || shouldShowGhostSection)
      "
      class="space-y-2"
    >
      <div class="border-t -mx-3" />
      <NTooltip v-if="selectedSpec" :showArrow="false">
        <template #trigger>
          <p class="textinfolabel">
            {{ $t("plan.options.self") }}
            <span class="opacity-80">
              ({{
                targetsForSpec(selectedSpec).length === flattenSpecCount
                  ? $t("plan.options.applies-to-all-tasks")
                  : $t("plan.options.applies-to-some-tasks", {
                      count: targetsForSpec(selectedSpec).length,
                      total: flattenSpecCount,
                    })
              }})
            </span>
          </p>
        </template>
        {{ $t("plan.options.split-into-multiple-issues-tip") }}
      </NTooltip>
      <PreBackupSection ref="preBackupSectionRef" />
      <GhostSection v-if="shouldShowGhostSection" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureBadge } from "@/components/FeatureGuard";
import { targetsForSpec } from "@/components/Plan";
import { GhostSection } from "@/components/Plan/components/Configuration";
import { provideGhostSettingContext } from "@/components/Plan/components/Configuration/GhostSection/context";
import { ApprovalFlowSection } from "@/components/Plan/components/IssueReviewView/Sidebar/ApprovalFlowSection";
import { issueServiceClientConnect } from "@/grpcweb";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { UpdateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { Issue } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";
import {
  convertNewIssueToOld,
  convertOldIssueToNew,
} from "@/utils/v1/issue-conversions";
import { specForTask, useIssueContext } from "../../logic";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReleaseInfo from "./ReleaseInfo.vue";
import TaskCheckSummarySection from "./TaskCheckSummarySection";

const { t } = useI18n();
const { isCreating, selectedTask, issue, events, allowChange } =
  useIssueContext();
const { project } = useCurrentProjectV1();
const preBackupSectionRef = ref<InstanceType<typeof PreBackupSection>>();

const selectedSpec = computed(() =>
  specForTask(issue.value.planEntity as Plan, selectedTask.value)
);

const flattenSpecCount = computed(
  () =>
    issue.value.planEntity?.specs.reduce(
      (acc, spec) => acc + targetsForSpec(spec).length,
      0
    ) || 0
);

const { shouldShow: shouldShowGhostSection, events: ghostEvents } =
  provideGhostSettingContext({
    isCreating,
    project,
    plan: computed(() => issue.value.planEntity as Plan),
    selectedSpec,
    selectedTask: selectedTask,
    issue,
  });

const shouldShowPreBackupSection = computed(() => {
  return preBackupSectionRef.value?.shouldShow ?? false;
});

const onIssueLabelsUpdate = async (labels: string[]) => {
  if (isCreating.value) {
    issue.value.labels = labels;
  } else {
    const issuePatch = Issue.fromPartial({
      ...issue.value,
      labels,
    });
    const newIssuePatch = convertOldIssueToNew(issuePatch);
    const request = create(UpdateIssueRequestSchema, {
      issue: newIssuePatch,
      updateMask: { paths: ["labels"] },
    });
    const newUpdated = await issueServiceClientConnect.updateIssue(request);
    const updated = convertNewIssueToOld(newUpdated);
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};

ghostEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});
</script>
