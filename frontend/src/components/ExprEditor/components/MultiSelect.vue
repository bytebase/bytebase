<template>
  <NSelect
    :value="value"
    :options="options"
    :multiple="true"
    :consistent-menu-width="false"
    :placeholder="$t('custom-approval.security-rule.condition.select-value')"
    :disabled="!allowAdmin"
    max-tag-count="responsive"
    size="small"
    style="min-width: 12rem; width: auto; overflow-x: hidden"
    @update:value="$emit('update:value', $event)"
  >
    <template #action>
      <NCheckbox
        :label="$t('common.all')"
        v-bind="checkAllState"
        @update:checked="toggleCheckAll"
      />
    </template>
  </NSelect>
</template>

<script lang="ts" setup>
import { NCheckbox, NSelect } from "naive-ui";
import { computed, toRef, watch } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { useSelectOptions } from "./common";

const props = defineProps<{
  value: string[] | number[];
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[] | number[]): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;

const options = useSelectOptions(toRef(props, "expr"));

const checkAllState = computed(() => {
  const selected = new Set<any>(props.value);
  const checked = options.value.every((opt) => {
    return selected.has(opt.value);
  });
  const indeterminate = props.value.length > 0 && !checked;
  return {
    checked,
    indeterminate,
  };
});

const toggleCheckAll = (on: boolean) => {
  if (on) {
    emit(
      "update:value",
      options.value.map((opt) => opt.value as string)
    );
  } else {
    emit("update:value", []);
  }
};

watch(
  [options, () => props.value],
  () => {
    const values = new Set(options.value.map((opt) => opt.value));
    const filtered = (props.value as any[]).filter((v) => values.has(v));
    if (filtered.length !== props.value.length) {
      // Some values are not suitable for the select options.
      emit("update:value", filtered);
    }
  },
  { immediate: true }
);
</script>
