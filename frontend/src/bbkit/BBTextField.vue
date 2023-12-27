<template>
  <NInput
    ref="inputField"
    v-bind="$attrs"
    v-model:value="state.text"
    :type="type"
    class="!border-none"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="placeholder"
    :autofocus="focusOnMount"
    :autosize="autosize"
    :status="state.hasError ? 'error' : undefined"
    @blur="onBlur"
    @update:value="onInput($event)"
    @keypress.enter="onPressEnter"
    @input="$emit('input', $event)"
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
    autosize?: boolean | { minRows?: number; maxRows?: number };
    type?: "text" | "password" | "textarea";
    required?: boolean;
    value?: string;
    placeholder?: string;
    disabled?: boolean;
    focusOnMount?: boolean;
    endsOnEnter?: boolean;
    clearable?: boolean;
  }>(),
  {
    autosize: false,
    type: "text",
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
  (event: "input", value: string): void;
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
