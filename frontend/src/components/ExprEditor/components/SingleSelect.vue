<template>
  <NSelect
    :value="value"
    :remote="optionConfig.remote"
    :options="state.rawOptionList"
    :loading="state.loading"
    :consistent-menu-width="false"
    :filterable="true"
    :placeholder="$t('cel.condition.select-value')"
    :disabled="readonly"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    @search="handleSearch"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { reactive, toRef, watch } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import { useExprEditorContext } from "../context";
import { useSelectOptionConfig } from "./common";

interface LocalState {
  loading: boolean;
  rawOptionList: SelectOption[];
}

const props = defineProps<{
  value: string | number;
  expr: ConditionExpr;
}>();

defineEmits<{
  (event: "update:value", value: string | number): void;
}>();

const context = useExprEditorContext();
const { readonly } = context;
const state = reactive<LocalState>({
  loading: false,
  rawOptionList: [],
});

const { optionConfig, factor } = useSelectOptionConfig(toRef(props, "expr"));

// Non-debounced function for initial load
const loadOptions = async (search: string) => {
  if (!optionConfig.value.search) {
    state.rawOptionList = [...optionConfig.value.options];
    return;
  }

  state.loading = true;
  try {
    const options = await optionConfig.value.search(search);
    state.rawOptionList = [...options];
  } finally {
    state.loading = false;
  }
};

// Debounced version for user-typed searches
const handleSearch = useDebounceFn(loadOptions, DEBOUNCE_SEARCH_DELAY);

watch(
  () => factor.value,
  async () => {
    // Use non-debounced loadOptions for initial load to ensure options
    // are available immediately when the component renders
    await loadOptions("");
    const search = optionConfig.value.search;
    if (!search) {
      return;
    }

    if (
      props.value &&
      !state.rawOptionList.find((opt) => opt.value === props.value)
    ) {
      const existed = new Set(state.rawOptionList.map((opt) => opt.value));
      const options = await search(props.value as string);
      for (const option of options) {
        if (!existed.has(option.value)) {
          state.rawOptionList.push(option);
          existed.add(option.value);
        }
      }
    }
  },
  { immediate: true }
);
</script>
