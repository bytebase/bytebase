<template>
  <RemoteResourceSelector
    v-if="optionConfig.search"
    size="small"
    max-tag-count="responsive"
    style="min-width: 12rem; width: auto; max-width: 20rem; overflow-x: hidden"
    :multiple="true"
    :disabled="readonly"
    :value="value"
    :search="optionConfig.search"
    :additional-options="additionalOptions"
    :consistent-menu-width="false"
    :fallback-option="optionConfig.fallback"
    @update:value="$emit('update:value', $event as string[])"
    @open="initial"
  />
  <NSelect
    v-else
    :value="value"
    :options="optionConfig.options"
    :multiple="true"
    :filterable="true"
    :consistent-menu-width="false"
    :clear-filter-after-select="false"
    :placeholder="$t('cel.condition.select-value')"
    :disabled="readonly"
    max-tag-count="responsive"
    size="small"
    style="min-width: 12rem; width: auto; max-width: 20rem; overflow-x: hidden"
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
import { computed, onMounted, ref, toRef } from "vue";
import RemoteResourceSelector from "@/components/v2/Select/RemoteResourceSelector/index.vue";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import { type ConditionExpr } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { initOptions, useSelectOptionConfig } from "./common";

const props = defineProps<{
  value: string[];
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[]): void;
}>();

const context = useExprEditorContext();
const { readonly } = context;
const { optionConfig } = useSelectOptionConfig(toRef(props, "expr"));
const additionalOptions = ref<ResourceSelectOption<unknown>[]>([]);

const initial = async () => {
  if (props.value.length === 0) {
    return;
  }
  const options = await initOptions(
    props.value.filter(
      (v) => !additionalOptions.value.find((option) => option.value === v)
    ),
    optionConfig.value
  );
  additionalOptions.value = options;
};

onMounted(async () => {
  await initial();
});

const checkAllState = computed(() => {
  const selected = new Set<string>(props.value);
  const checked =
    selected.size > 0 &&
    optionConfig.value.options.every((opt) => {
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
      optionConfig.value.options.map((opt) => opt.value as string)
    );
  } else {
    emit("update:value", []);
  }
};
</script>
