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
  projectOfIssue,
} from "@/components/IssueV1/logic";
import { PreBackupSection } from "@/components/Plan/components/Sidebar";
import { providePreBackupSettingContext } from "@/components/Plan/components/Sidebar/PreBackupSection/context";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import type { Plan } from "@/types/proto/v1/plan_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import TaskRollbackSection from "./TaskRollbackSection.vue";
import { ROLLBACK_AVAILABLE_ENGINES } from "./common";

const { isCreating, issue, selectedTask, events } = useIssueContext();
const {
  shouldShow: shouldShowPreBackupSection,
  enabled: preBackupEnabled,
  events: preBackupEvents,
} = providePreBackupSettingContext({
  isCreating,
  project: computed(() => projectOfIssue(issue.value)),
  plan: computed(() => issue.value.planEntity as Plan),
  selectedSpec: computed(() =>
    specForTask(issue.value.planEntity, selectedTask.value)
  ),
  selectedTask,
  issue,
  rollout: computed(() => issue.value.rolloutEntity),
});

const database = computed(() =>
  databaseForTask(projectOfIssue(issue.value), selectedTask.value)
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
</script>
