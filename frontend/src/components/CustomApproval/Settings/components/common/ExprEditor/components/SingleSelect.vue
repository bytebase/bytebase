<template>
  <NSelect
    :value="value"
    :options="options"
    :consistent-menu-width="false"
    :placeholder="$t('custom-approval.security-rule.condition.select-value')"
    :disabled="!allowAdmin"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { toRef, watch } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { useSelectOptions } from "./common";

const props = defineProps<{
  value: string | number;
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string | number): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;

const options = useSelectOptions(toRef(props, "expr"));

watch(
  [options, () => props.value],
  () => {
    if (options.value.length === 0) return;
    if (!options.value.find((opt) => opt.value === props.value)) {
      emit("update:value", options.value[0].value!);
    }
  },
  { immediate: true }
);
</script>
