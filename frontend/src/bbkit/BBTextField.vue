<template>
  <input
    ref="inputField"
    v-model="state.text"
    type="text"
    autocomplete="off"
    class="text-main rounded-md placeholder:text-control-placeholder"
    :class="
      state.hasError
        ? 'border-error focus:ring-error focus:border-error'
        : bordered
        ? 'border-control-border focus:ring-control focus:border-control disabled:bg-gray-50'
        : 'border-transparent focus:ring-control focus:border-control disabled:text-control'
    "
    :disabled="disabled"
    :placeholder="placeholder"
    @focus="onFocus"
    @blur="onBlur"
    @input="onInput($event)"
    @keypress.enter="onPressEnter"
  />
</template>

<script lang="ts" setup>
import { isEmpty } from "lodash-es";
import { nextTick, onMounted, reactive, ref, watch, withDefaults } from "vue";

interface LocalState {
  text: string;
  originalText: string;
  hasError: boolean;
}

const props = withDefaults(
  defineProps<{
    required?: boolean;
    forceRequired?: boolean;
    value?: string;
    placeholder?: string;
    disabled?: boolean;
    bordered?: boolean;
    focusOnMount?: boolean;
    endsOnEnter?: boolean;
  }>(),
  {
    required: false,
    forceRequired: true,
    value: "",
    placeholder: "",
    disabled: false,
    bordered: true,
    focusOnMount: false,
    endsOnEnter: false,
  }
);

const emit = defineEmits<{
  (event: "end-editing", value: string): void;
  (event: "input", e: Event): void;
}>();

const inputField = ref();

const state = reactive<LocalState>({
  text: props.value,
  originalText: "",
  hasError: false,
});

onMounted(() => {
  if (props.focusOnMount) {
    inputField.value.focus();
    inputField.value.select();
  }
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

const onFocus = () => {
  state.originalText = state.text;
};

const onBlur = () => {
  if (props.required && isEmpty(state.text.trim())) {
    state.hasError = true;
    nextTick(() => {
      if (props.forceRequired && inputField.value) {
        state.text = state.originalText;
        // Since we set focus in the nextTick, inputField might already disappear due to outside state change.
        inputField.value.focus();
        nextTick(() => {
          inputField.value.select();
        });
      }
    });
  } else {
    state.hasError = false;
    emit("end-editing", state.text);
  }
};

const onInput = (e: Event) => {
  state.hasError = false;
  emit("input", e);
};

const onPressEnter = (e: Event) => {
  if (props.endsOnEnter) {
    const input = e.target as HTMLInputElement;
    input?.blur();
  }
};
</script>
