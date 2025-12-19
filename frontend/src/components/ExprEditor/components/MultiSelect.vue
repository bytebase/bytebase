<template>
  <NSelect
    :value="value"
    :remote="optionConfig.remote"
    :options="state.rawOptionList"
    :loading="state.loading"
    :multiple="true"
    :filterable="true"
    :consistent-menu-width="false"
    :clear-filter-after-select="false"
    :placeholder="$t('cel.condition.select-value')"
    :disabled="readonly"
    max-tag-count="responsive"
    size="small"
    style="min-width: 12rem; width: auto; max-width: 20rem; overflow-x: hidden"
    @search="handleSearch"
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
import { useDebounceFn } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NCheckbox, NSelect } from "naive-ui";
import { computed, reactive, toRef, watch } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import { useExprEditorContext } from "../context";
import { useSelectOptionConfig } from "./common";

interface LocalState {
  loading: boolean;
  rawOptionList: SelectOption[];
}

const props = defineProps<{
  value: string[] | number[];
  expr: ConditionExpr;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string[] | number[]): void;
}>();

const context = useExprEditorContext();
const { readonly } = context;
const state = reactive<LocalState>({
  loading: false,
  rawOptionList: [],
});

const { optionConfig, factor } = useSelectOptionConfig(toRef(props, "expr"));

const checkAllState = computed(() => {
  const selected = new Set<any>(props.value);
  const checked =
    selected.size > 0 &&
    state.rawOptionList.every((opt) => {
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
      state.rawOptionList.map((opt) => opt.value as string)
    );
  } else {
    emit("update:value", []);
  }
};

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
    // valid initial value.
    const search = optionConfig.value.search;
    if (!search) {
      return;
    }
    const existed = new Set(state.rawOptionList.map((opt) => opt.value));
    const pendingInit = props.value.filter((val) => !existed.has(val));
    for (const pending of pendingInit) {
      const options = await search(pending as string);
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
