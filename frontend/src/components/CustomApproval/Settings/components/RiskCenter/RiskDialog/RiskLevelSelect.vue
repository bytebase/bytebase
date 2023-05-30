<template>
  <NSelect
    style="width: 8rem"
    :value="value"
    :options="options"
    :placeholder="$t('custom-approval.security-rule.risk.select')"
    :consistent-menu-width="false"
    :disabled="disabled || !allowAdmin"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NSelect, SelectOption, type SelectProps } from "naive-ui";

import { PresetRiskLevelList } from "@/types";
import { useRiskCenterContext } from "../context";
import { levelText } from "../../common";

export interface RiskLevelSelectorProps extends SelectProps {
  value: number;
  disabled?: boolean;
}

defineProps<RiskLevelSelectorProps>();

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const context = useRiskCenterContext();
const { allowAdmin } = context;

const options = computed(() => {
  return PresetRiskLevelList.map<SelectOption>(({ level }) => ({
    label: levelText(level),
    value: level,
  }));
});
</script>
