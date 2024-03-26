<template>
  <NSelect
    style="width: 11rem"
    :value="value"
    :options="options"
    :placeholder="$t('custom-approval.risk-rule.source.select')"
    :consistent-menu-width="false"
    :disabled="disabled || !allowAdmin"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect, type SelectProps } from "naive-ui";
import { computed } from "vue";
import { SupportedSourceList } from "@/types";
import type { Risk_Source } from "@/types/proto/v1/risk_service";
import { sourceText } from "../../common";
import { useRiskCenterContext } from "../context";

export interface RiskSourceSelectProps extends /* @vue-ignore */ SelectProps {
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
