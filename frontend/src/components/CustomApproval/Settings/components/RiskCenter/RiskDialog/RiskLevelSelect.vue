<template>
  <NSelect
    style="width: 11rem"
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

export interface RiskSelectorProps extends SelectProps {
  value: number;
  disabled?: boolean;
}

defineProps<RiskSelectorProps>();

defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const context = useRiskCenterContext();
const { allowAdmin } = context;

const options = computed(() => {
  return PresetRiskLevelList.map<SelectOption>(({ name, level }) => ({
    label: name,
    value: level,
  }));
});
</script>
