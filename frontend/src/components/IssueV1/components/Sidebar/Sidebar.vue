<template>
  <div class="flex flex-col gap-y-3 py-2 px-3">
    <ReleaseInfo />
    <TaskCheckSummarySection />
    <ReviewSection />
    <IssueLabels />

    <template
      v-if="
        selectedSpec && (shouldShowPreBackupSection || shouldShowGhostSection)
      "
    >
      <div class="border-t -mx-3" />
      <NTooltip :showArrow="false">
        <template #trigger>
          <p class="textinfolabel -mb-2">
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
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { targetsForSpec } from "@/components/Plan";
import { GhostSection } from "@/components/Plan/components/Configuration";
import { provideGhostSettingContext } from "@/components/Plan/components/Configuration/GhostSection/context";
import { useCurrentProjectV1 } from "@/store";
import type { Plan } from "@/types/proto/v1/plan_service";
import { specForTask, useIssueContext } from "../../logic";
import IssueLabels from "./IssueLabels.vue";
import PreBackupSection from "./PreBackupSection";
import ReleaseInfo from "./ReleaseInfo.vue";
import ReviewSection from "./ReviewSection";
import TaskCheckSummarySection from "./TaskCheckSummarySection";

const { isCreating, selectedTask, issue, events } = useIssueContext();
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

ghostEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});
</script>
