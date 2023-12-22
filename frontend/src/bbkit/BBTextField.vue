<template>
  <NInput
    class="!border-none"
    ref="inputField"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="placeholder"
    v-model:value="state.text"
    :autofocus="focusOnMount"
    :status="state.hasError ? 'error' : undefined"
    @blur="onBlur"
    @input="onInput($event)"
    @keypress.enter="onPressEnter"
  />
</template>

<script lang="ts" setup>
import { isEmpty } from "lodash-es";
import { NInput } from "naive-ui";
import { reactive, ref, watch, withDefaults } from "vue";

interface LocalState {
  text: string;
  hasError: boolean;
}

const props = withDefaults(
  defineProps<{
    required?: boolean;
    value?: string;
    placeholder?: string;
    disabled?: boolean;
    focusOnMount?: boolean;
    endsOnEnter?: boolean;
    clearable?: boolean;
  }>(),
  {
    required: false,
    value: "",
    placeholder: "",
    disabled: false,
    focusOnMount: false,
    endsOnEnter: false,
    clearable: false,
  }
);

const emit = defineEmits<{
  (event: "end-editing", value: string): void;
  (event: "update:value", value: string): void;
  (event: "input", e: Event): void;
}>();

const inputField = ref();

const state = reactive<LocalState>({
  text: props.value,
  hasError: false,
});

watch(
  () => props.value,
  (cur) => {
    state.text = cur;
    if (props.required && isEmpty(state.text.trim())) {
      state.hasError = true;
    } else {
      state.hasError = false;
    }
  }
);

const onBlur = () => {
  if (props.required && isEmpty(state.text.trim())) {
    state.hasError = true;
  } else {
    state.hasError = false;
    emit("end-editing", state.text);
  }
};

const onInput = (value: string) => {
  state.hasError = false;
  emit("update:value", value);
};

const onPressEnter = (e: Event) => {
  if (props.endsOnEnter) {
    const input = e.target as HTMLInputElement;
    input?.blur();
  }
};
</script>
