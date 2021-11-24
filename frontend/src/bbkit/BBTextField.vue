<template>
  <input
    ref="inputField"
    v-model="state.text"
    type="text"
    autocomplete="off"
    class="text-main rounded-md"
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
  />
</template>

<script lang="ts">
import { nextTick, onMounted, reactive, ref, watch } from "vue";
import isEmpty from "lodash-es/isEmpty";

interface LocalState {
  text: string;
  originalText: string;
  hasError: boolean;
}

export default {
  name: "BBTextField",
  props: {
    required: {
      default: false,
      type: Boolean,
    },
    value: {
      default: "",
      type: String,
    },
    placeholder: {
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    bordered: {
      default: true,
      type: Boolean,
    },
    focusOnMount: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["end-editing", "input"],
  setup(props, { emit }) {
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
      }
    );

    const onFocus = () => {
      state.originalText = state.text;
    };

    const onBlur = () => {
      if (props.required && isEmpty(state.text.trim())) {
        state.hasError = true;
        nextTick(() => {
          state.text = state.originalText;
          if (inputField.value) {
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

    const onInput = (e: any) => {
      state.hasError = false;
      emit("input", e);
    };

    return {
      inputField,
      state,
      onFocus,
      onBlur,
      onInput,
    };
  },
};
</script>
