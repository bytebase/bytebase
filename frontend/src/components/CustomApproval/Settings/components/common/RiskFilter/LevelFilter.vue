<template>
  <div class="flex items-center gap-x-3">
    <label
      v-for="item in riskLevelFilterItemList"
      :key="item.value"
      class="flex items-center gap-x-2 text-sm text-gray-600"
    >
      <NCheckbox
        :checked="isCheckedLevel(item.value)"
        @update:checked="toggleCheckLevel(item.value, $event)"
      >
        <BBBadge
          class="whitespace-nowrap"
          :text="item.label"
          :can-remove="false"
          :style="item.style"
          size="small"
        />
      </NCheckbox>
    </label>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NCheckbox } from "naive-ui";

import BBBadge, { type BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import { PresetRiskLevel, PresetRiskLevelList } from "@/types";
import { levelText } from "../../common";
import { useRiskFilter } from "./context";

type RiskLevelFilterItem = {
  value: number;
  label: string;
  style: BBBadgeStyle;
};

const { levels } = useRiskFilter();

const riskLevelFilterItemList = computed(() => {
  return PresetRiskLevelList.map<RiskLevelFilterItem>(({ level }) => {
    return {
      value: level,
      label: levelText(level),
      style:
        level === PresetRiskLevel.HIGH
          ? "CRITICAL"
          : level === PresetRiskLevel.MODERATE
          ? "WARN"
          : level === PresetRiskLevel.LOW
          ? "INFO"
          : "DISABLED",
    };
  });
});

const isCheckedLevel = (level: number) => {
  return levels.value.has(level);
};

const toggleCheckLevel = (level: number, checked: boolean) => {
  if (checked) levels.value.add(level);
  else levels.value.delete(level);
};
</script>
