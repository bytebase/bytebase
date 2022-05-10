<template>
  <div>
    <PipelineStageList>
      <template #task-name-of-stage="{ stage }">
        {{ taskNameOfStage(stage) }}
      </template>
    </PipelineStageList>
  </div>
</template>

<script lang="ts" setup>
import type { Stage, StageCreate } from "@/types";
import { activeTaskInStage } from "@/utils";
import { useIssueContext } from "./context";

const { create } = useIssueContext();

const taskNameOfStage = (stage: Stage | StageCreate) => {
  if (create.value) {
    return stage.taskList[0].status;
  }
  const activeTask = activeTaskInStage(stage as Stage);
  const { taskList } = stage as Stage;
  for (let i = 0; i < stage.taskList.length; i++) {
    if (taskList[i].id == activeTask.id) {
      return `${activeTask.name} (${i + 1}/${stage.taskList.length})`;
    }
  }
  return activeTask.name;
};
</script>
