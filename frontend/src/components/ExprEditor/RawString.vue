<!-- eslint-disable vue/no-mutating-props -->

<template>
  <NInputGroup class="w-full flex items-start overflow-x-hidden">
    <NInput
      v-model:value="expr.content"
      type="textarea"
      :autosize="{ minRows: 2, maxRows: 4 }"
      placeholder="Enter raw CEL expression"
      size="small"
    />

    <NButton
      v-if="allowAdmin"
      size="small"
      type="default"
      :style="'flex-shrink: 0;padding-left: 0;padding-right: 0;--n-width: 28px;--n-color: white;'"
      @click="$emit('remove')"
    >
      <heroicons:trash class="w-3.5 h-3.5" />
    </NButton>
  </NInputGroup>
</template>

<script lang="ts" setup>
import { NButton, NInput, NInputGroup } from "naive-ui";
import { watch } from "vue";
import { type RawStringExpr } from "@/plugins/cel";
import { useExprEditorContext } from "./context";

const props = defineProps<{
  expr: RawStringExpr;
}>();

const emit = defineEmits<{
  (event: "remove"): void;
  (event: "update"): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;

watch(
  () => props.expr,
  () => {
    emit("update");
  },
  { deep: true }
);
</script>
