<template>
  <div class="flex items-center gap-x-4 px-4 py-2">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("issue.sql-check.sql-checks") }}
    </div>

    <SQLCheckButton
      v-if="database"
      :key="selectedSpec.id"
      :get-statement="getStatement"
      :database="database"
      :change-type="changeType"
      :button-props="{
        size: 'tiny',
      }"
      button-style="--n-padding: 0 8px 0 6px; --n-icon-margin: 3px;"
      class="justify-between flex-1"
      @update:advices="$emit('update:advices', $event)"
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
import { databaseForSpec } from "@/components/Plan/logic";
import { SQLCheckButton } from "@/components/SQLCheck";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import { Release_File_ChangeType } from "@/types/proto/v1/release_service";
import { Advice } from "@/types/proto/v1/sql_service";
import { usePlanContext } from "../../logic";
import { useSpecSheet } from "../StatementSection/useSpecSheet";
import SQLCheckBadge from "./SQLCheckBadge.vue";

defineEmits<{
  (event: "update:advices", advices: Advice[] | undefined): void;
}>();

const { plan, selectedSpec } = usePlanContext();
const { sheetStatement } = useSpecSheet();

const database = computed(() => {
  return databaseForSpec(plan.value, selectedSpec.value);
});

const getStatement = async () => {
  return {
    statement: sheetStatement.value,
    errors: [],
  };
};

const changeType = computed((): Release_File_ChangeType | undefined => {
  switch (selectedSpec.value?.changeDatabaseConfig?.type) {
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
