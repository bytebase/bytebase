<template>
  <NSelect
    :value="value"
    :remote="optionConfig.remote"
    :options="state.rawOptionList"
    :loading="state.loading"
    :consistent-menu-width="false"
    :filterable="true"
    :placeholder="$t('cel.condition.select-value')"
    :disabled="!allowAdmin"
    size="small"
    style="min-width: 7rem; width: auto; overflow-x: hidden"
    @search="handleSearch"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NSelect } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { toRef, watch, reactive } from "vue";
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

const emit = defineEmits<{
  (event: "update:value", value: string | number): void;
}>();

const context = useExprEditorContext();
const { allowAdmin } = context;
const state = reactive<LocalState>({
  loading: false,
  rawOptionList: [],
});

const { optionConfig, factor } = useSelectOptionConfig(toRef(props, "expr"));

const handleSearch = useDebounceFn(async (search: string) => {
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
}, DEBOUNCE_SEARCH_DELAY);

watch(
  () => factor.value,
  async () => {
    await handleSearch("");
    if (state.rawOptionList.length === 0) {
      return;
    }
    if (
      !props.value ||
      !state.rawOptionList.find((opt) => opt.value === props.value)
    ) {
      emit("update:value", state.rawOptionList[0].value!);
    }
  },
  { immediate: true }
);
</script>
