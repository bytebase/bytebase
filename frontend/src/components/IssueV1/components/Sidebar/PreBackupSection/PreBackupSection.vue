<template>
  <TaskRollbackSection v-if="shouldShowTaskRollbackSection" />
  <PreBackupSection v-else-if="shouldShowPreBackupSection" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  latestTaskRunForTask,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { PreBackupSection } from "@/components/Plan/components/Configuration";
import { providePreBackupSettingContext } from "@/components/Plan/components/Configuration/PreBackupSection/context";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { useCurrentProjectV1 } from "@/store";
import type { Plan } from "@/types/proto/v1/plan_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import TaskRollbackSection from "./TaskRollbackSection.vue";
import { ROLLBACK_AVAILABLE_ENGINES } from "./common";

const { isCreating, issue, selectedTask, events } = useIssueContext();
const { project } = useCurrentProjectV1();
const {
  shouldShow: shouldShowPreBackupSection,
  enabled: preBackupEnabled,
  events: preBackupEvents,
} = providePreBackupSettingContext({
  isCreating,
  project,
  plan: computed(() => issue.value.planEntity as Plan),
  selectedSpec: computed(() =>
    specForTask(issue.value.planEntity as Plan, selectedTask.value)
  ),
  selectedTask,
  issue,
  rollout: computed(() => issue.value.rolloutEntity),
});

const database = computed(() =>
  databaseForTask(project.value, selectedTask.value)
);

const latestTaskRun = computed(() =>
  latestTaskRunForTask(issue.value, selectedTask.value)
);

const shouldShowTaskRollbackSection = computed((): boolean => {
  if (!shouldShowPreBackupSection.value) {
    return false;
  }
  if (!preBackupEnabled.value) {
    return false;
  }
  if (
    !ROLLBACK_AVAILABLE_ENGINES.includes(database.value.instanceResource.engine)
  ) {
    return false;
  }
  if (!latestTaskRun.value) {
    return false;
  }
  if (latestTaskRun.value.status !== TaskRun_Status.DONE) {
    return false;
  }
  if (latestTaskRun.value.priorBackupDetail?.items.length === 0) {
    return false;
  }
  return true;
});

preBackupEvents.on("update", () => {
  events.emit("status-changed", {
    eager: true,
  });
});

defineExpose({
  shouldShow: computed(
    () =>
      shouldShowTaskRollbackSection.value || shouldShowPreBackupSection.value
  ),
});
</script>
