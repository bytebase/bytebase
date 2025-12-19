<!-- eslint-disable vue/no-mutating-props -->

<template>
  <NInputGroup class="w-full flex items-center overflow-x-hidden">
    <FactorSelect :expr="expr" />

    <OperatorSelect :expr="expr" />

    <ValueInput :expr="expr" />

    <NButton
      size="small"
      type="default"
      :disabled="readonly"
      :style="'shrink: 0;padding-left: 0;padding-right: 0;--n-width: 28px;--n-color: white;'"
      @click="$emit('remove')"
    >
      <heroicons:trash class="w-3.5 h-3.5" />
    </NButton>
  </NInputGroup>
</template>

<script lang="ts" setup>
import { NButton, NInputGroup } from "naive-ui";
import { watch } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { FactorSelect, OperatorSelect, ValueInput } from "./components";
import { useExprEditorContext } from "./context";

const props = defineProps<{
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "remove"): void;
  (event: "update"): void;
}>();

const context = useExprEditorContext();
const { readonly } = context;

watch(
  () => props.expr,
  () => {
    emit("update");
  },
  { deep: true }
);
</script>
