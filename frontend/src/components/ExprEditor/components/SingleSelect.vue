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
import { toRef, watch, reactive, watchEffect } from "vue";
import { type ConditionExpr } from "@/plugins/cel";
import { useExprEditorContext } from "../context";
import { useSelectOptions } from "./common";

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

const optionConfig = useSelectOptions(toRef(props, "expr"));

watchEffect(() => {
  state.rawOptionList = [...optionConfig.value.options];
});

watch(
  [state.rawOptionList, () => props.value],
  () => {
    if (state.rawOptionList.length === 0) return;
    if (!state.rawOptionList.find((opt) => opt.value === props.value)) {
      emit("update:value", state.rawOptionList[0].value!);
    }
  },
  { immediate: true }
);

const handleSearch = useDebounceFn(async (search: string) => {
  if (!search || !optionConfig.value.search) {
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
}, 500);
</script>
