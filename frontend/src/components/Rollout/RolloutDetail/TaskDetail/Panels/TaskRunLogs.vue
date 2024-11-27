<template>
  <div v-for="taskRun in taskRuns" :key="taskRun.name" class="mb-4">
    <p class="text-lg">
      <HumanizeDate :date="getDateForPbTimestamp(taskRun.createTime)" />
    </p>
    <TaskRunLogTable
      :task-run="taskRun"
      :sheet="sheetStore.getSheetByName(sheetNameOfTaskV1(task))"
    />
  </div>
</template>

<script lang="ts" setup>
import TaskRunLogTable from "@/components/IssueV1/components/TaskRunSection/TaskRunLogTable/TaskRunLogTable.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { useSheetV1Store } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import { sheetNameOfTaskV1 } from "@/utils";
import { useTaskDetailContext } from "../context";

const { task, taskRuns } = useTaskDetailContext();
const sheetStore = useSheetV1Store();
</script>
