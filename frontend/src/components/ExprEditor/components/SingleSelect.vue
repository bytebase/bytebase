<template>
  <RemoteResourceSelector
    v-if="optionConfig.search"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    :multiple="false"
    :disabled="readonly"
    :value="value"
    :search="optionConfig.search"
    :consistent-menu-width="false"
    :additional-options="additionalOptions"
    :fallback-option="optionConfig.fallback"
    @update:value="$emit('update:value', $event as string)"
    @open="initial"
  />
  <NSelect
    v-else
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    :value="value"
    :options="optionConfig.options"
    :consistent-menu-width="false"
    :filterable="true"
    :placeholder="$t('cel.condition.select-value')"
    :disabled="readonly"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect } from "naive-ui";
import { onMounted, ref, toRef } from "vue";
import RemoteResourceSelector from "@/components/v2/Select/RemoteResourceSelector/index.vue";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import { type ConditionExpr } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { initOptions, useSelectOptionConfig } from "./common";

const props = defineProps<{
  value: string;
  expr: ConditionExpr;
}>();

defineEmits<{
  (event: "update:value", value: string): void;
}>();

const context = useExprEditorContext();
const { readonly } = context;

const { optionConfig } = useSelectOptionConfig(toRef(props, "expr"));
const additionalOptions = ref<ResourceSelectOption<unknown>[]>([]);

const initial = async () => {
  if (!props.value || additionalOptions.value[0]?.value === props.value) {
    return;
  }
  const options = await initOptions([props.value], optionConfig.value);
  additionalOptions.value = options;
};

onMounted(async () => {
  await initial();
});
</script>
