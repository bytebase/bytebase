<template>
  <div v-if="show" class="w-full flex items-center gap-x-4 px-4 py-2">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("issue.sql-check.sql-checks") }}
    </div>
    <div
      v-if="database"
      class="grow flex flex-row items-center justify-between gap-2"
    >
      <span v-if="checkResult === undefined" class="textinfolabel">
        {{ $t("issue.sql-check.not-executed-yet") }}
      </span>
      <div v-else class="flex flex-row justify-start items-center gap-2">
        <SQLCheckBadge :advices="checkResult.advices" />
        <NTooltip v-if="checkResult.affectedRows > 0">
          <template #trigger>
            <NTag round>
              <span class="opacity-80"
                >{{ $t("task.check-type.affected-rows.self") }}:
              </span>
              <span>{{ checkResult.affectedRows }}</span>
            </NTag>
          </template>
          {{ $t("task.check-type.affected-rows.description") }}
        </NTooltip>
      </div>

      <SQLCheckButton
        :key="selectedTask.name"
        :database="database"
        button-style="--n-padding: 0 8px 0 6px; --n-icon-margin: 3px;"
        :show-code-location="true"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import SQLCheckBadge from "./SQLCheckBadge.vue";
import SQLCheckButton from "./SQLCheckButton.vue";
import { useIssueSQLCheckContext } from "./context";

const { issue, selectedTask } = useIssueContext();

const { enabled, resultMap } = useIssueSQLCheckContext();

const database = computed(() => {
  return databaseForTask(issue.value.projectEntity, selectedTask.value);
});

const show = computed(() => {
  if (!enabled.value) {
    return false;
  }
  const type = selectedTask.value.type;
  if (type === Task_Type.DATABASE_SCHEMA_BASELINE) {
    return false;
  }
  if (TaskTypeListWithStatement.includes(type)) {
    return true;
  }
  return false;
});

const checkResult = computed(() => {
  const database = databaseForTask(
    issue.value.projectEntity,
    selectedTask.value
  );
  return resultMap.value[database.name] || undefined;
});
</script>
