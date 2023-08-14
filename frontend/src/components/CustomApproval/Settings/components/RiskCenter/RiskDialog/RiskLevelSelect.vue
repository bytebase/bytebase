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
import { NSelect, SelectOption, type SelectProps } from "naive-ui";
import { computed } from "vue";
import { PresetRiskLevelList } from "@/types";
import { levelText } from "../../common";
import { useRiskCenterContext } from "../context";

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
