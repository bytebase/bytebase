<template>
  <div v-if="show" class="flex items-center gap-x-4 px-4 py-2">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("issue.sql-check.sql-checks") }}
    </div>

    <SQLCheckButton
      v-if="database"
      :key="selectedTask.uid"
      :statement="sheetStatement"
      :database="database"
      :button-props="{
        size: 'tiny',
      }"
      button-style="--n-padding: 0 8px 0 6px; --n-icon-margin: 3px;"
      class="justify-between flex-1"
    >
      <template #result="{ advices, isRunning }">
        <template v-if="advices === undefined">
          <span class="textinfolabel">
            {{ $t("issue.sql-check.not-executed-yet") }}
          </span>
        </template>
        <SQLCheckBadge v-else :is-running="isRunning" :advices="advices" />
      </template>
    </SQLCheckButton>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useTaskSheet } from "@/components/IssueV1/components/StatementSection/useTaskSheet";
import { useIssueContext, databaseForTask } from "@/components/IssueV1/logic";
import { SQLCheckButton } from "@/components/SQLCheck";
import { TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import SQLCheckBadge from "./SQLCheckBadge.vue";

const { issue, selectedTask } = useIssueContext();
const { sheetStatement } = useTaskSheet();
const database = computed(() => {
  return databaseForTask(issue.value, selectedTask.value);
});

const show = computed(() => {
  const type = selectedTask.value.type;
  if (type === Task_Type.DATABASE_SCHEMA_BASELINE) {
    return false;
  }
  if (TaskTypeListWithStatement.includes(type)) {
    return true;
  }
  return false;
});
</script>
