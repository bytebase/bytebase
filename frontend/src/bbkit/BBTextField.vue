<template>
  <input
    type="text"
    ref="inputField"
    autocomplete="off"
    class="text-main rounded-md disabled:bg-gray-50 disabled:cursor-not-allowed"
    :class="
      state.hasError
        ? 'border-error focus:ring-error focus:border-error'
        : bordered
        ? 'border-control-border focus:ring-control focus:border-control'
        : 'border-transparent focus:ring-control focus:border-control'
    "
    v-model="state.text"
    :disabled="disabled"
    :placeholder="placeholder"
    @focus="onFocus"
    @blur="onBlur"
    @input="onInput"
  />
</template>

<script lang="ts">
import { computed, nextTick, reactive, ref, watch } from "vue";
import isEmpty from "lodash-es/isEmpty";

interface LocalState {
  text: string;
  originalText: string;
  hasError: boolean;
}

export default {
  name: "BBTextField",
  emits: ["end-editing"],
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
  },
  setup(props, { emit }) {
    const inputField = ref<HTMLInputElement>();

    const state = reactive<LocalState>({
      text: props.value,
      originalText: "",
      hasError: false,
    });

    watch(
      () => props.value,
      (cur, _) => {
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
          inputField.value!.focus();
          nextTick(() => {
            inputField.value!.select();
          });
        });
      } else {
        state.hasError = false;
        emit("end-editing", state.text);
      }
    };

    const onInput = () => {
      state.hasError = false;
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
