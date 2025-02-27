<template>
  <div v-if="show" class="flex items-center gap-x-4 px-4 py-2">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("issue.sql-check.sql-checks") }}
    </div>

    <SQLCheckButton
      v-if="database"
      :key="selectedTask.name"
      :get-statement="getStatement"
      :database="database"
      :change-type="changeType"
      :button-props="{
        size: 'small',
      }"
      button-style="--n-padding: 0 8px 0 6px; --n-icon-margin: 3px;"
      class="justify-between flex-1"
      :show-code-location="true"
      @update:advices="$emit('update:advices', $event)"
    >
      <template #result="{ affectedRows, advices, isRunning }">
        <span v-if="advices === undefined" class="textinfolabel">
          {{ $t("issue.sql-check.not-executed-yet") }}
        </span>
        <div v-else class="flex flex-row justify-start items-center gap-2">
          <SQLCheckBadge :is-running="isRunning" :advices="advices" />
          <NTooltip v-if="affectedRows && affectedRows > 0">
            <template #trigger>
              <NTag round>
                <span class="opacity-80"
                  >{{ $t("task.check-type.affected-rows.self") }}:
                </span>
                <span>{{ affectedRows }}</span>
              </NTag>
            </template>
            {{ $t("task.check-type.affected-rows.description") }}
          </NTooltip>
        </div>
      </template>
    </SQLCheckButton>
  </div>
</template>

<script lang="ts" setup>
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import {
  useIssueContext,
  databaseForTask,
  specForTask,
} from "@/components/IssueV1/logic";
import { SQLCheckButton } from "@/components/SQLCheck";
import { TaskTypeListWithStatement } from "@/types";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import { Release_File_ChangeType } from "@/types/proto/v1/release_service";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { Advice } from "@/types/proto/v1/sql_service";
import { useTaskSheet } from "../StatementSection/useTaskSheet";
import SQLCheckBadge from "./SQLCheckBadge.vue";

defineEmits<{
  (event: "update:advices", advices: Advice[] | undefined): void;
}>();

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

const getStatement = async () => {
  return {
    statement: sheetStatement.value,
    errors: [],
  };
};

const changeType = computed((): Release_File_ChangeType | undefined => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  switch (spec?.changeDatabaseConfig?.type) {
    case Plan_ChangeDatabaseConfig_Type.MIGRATE:
      return Release_File_ChangeType.DDL;
    case Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
      return Release_File_ChangeType.DDL_GHOST;
    case Plan_ChangeDatabaseConfig_Type.DATA:
      return Release_File_ChangeType.DML;
  }
  return undefined;
});
</script>
