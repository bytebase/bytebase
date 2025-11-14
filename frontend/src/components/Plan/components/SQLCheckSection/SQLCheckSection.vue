<template>
  <div class="w-full flex items-center gap-x-4 px-4 py-2">
    <div class="textlabel h-[26px] inline-flex items-center">
      {{ $t("issue.sql-check.sql-checks") }}
    </div>
    <div
      v-if="isValidDatabaseName(database.name)"
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
        :key="database.name"
        button-style="--n-padding: 0 8px 0 6px; --n-icon-margin: 3px;"
        :show-code-location="true"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { isValidDatabaseName } from "@/types";
import { usePlanSQLCheckContext } from "./context";
import SQLCheckBadge from "./SQLCheckBadge.vue";
import SQLCheckButton from "./SQLCheckButton.vue";

const { database, resultMap } = usePlanSQLCheckContext();

const checkResult = computed(() => {
  const result = resultMap.value[database.value.name] || undefined;
  if (!result) return undefined;

  return result;
});
</script>
