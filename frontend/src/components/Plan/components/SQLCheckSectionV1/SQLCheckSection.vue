<template>
  <div class="w-full flex items-center gap-x-4 px-4 py-2">
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
        :key="selectedSpec.id"
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
import { databaseForSpec, usePlanContext } from "../../logic";
import SQLCheckBadge from "./SQLCheckBadge.vue";
import SQLCheckButton from "./SQLCheckButton.vue";
import { usePlanSQLCheckContext } from "./context";

const { plan, selectedSpec } = usePlanContext();

const { resultMap } = usePlanSQLCheckContext();

const database = computed(() => {
  return databaseForSpec(plan.value.projectEntity, selectedSpec.value);
});

const checkResult = computed(() => {
  return resultMap.value[database.value.name] || undefined;
});
</script>
