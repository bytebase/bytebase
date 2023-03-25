<template>
  <NSelect
    style="width: 11rem"
    :value="value"
    :options="options"
    :placeholder="$t('custom-approval.security-rule.source.select')"
    :consistent-menu-width="false"
    :disabled="disabled || !allowAdmin"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NSelect, SelectOption, type SelectProps } from "naive-ui";

import { Risk_Source } from "@/types/proto/v1/risk_service";
import { SupportedSourceList } from "@/types";
import { useRiskCenterContext } from "../context";
import { sourceText } from "../../common";

export interface RiskSourceSelectProps extends SelectProps {
  value: Risk_Source;
  disabled?: boolean;
}

defineProps<RiskSourceSelectProps>();

defineEmits<{
  (event: "update:value", source: Risk_Source | undefined): void;
}>();

const context = useRiskCenterContext();
const { allowAdmin } = context;

const options = computed(() => {
  return SupportedSourceList.map<SelectOption>((source) => ({
    label: sourceText(source),
    value: source,
  }));
});
</script>
