<template>
  <input
    type="text"
    ref="inputField"
    autocomplete="off"
    class="text-main sm:text-sm rounded-md focus:ring-control focus:border-control"
    :class="hasError ? 'border-error' : 'border-control-border'"
    v-model="value"
    :placeholder="placeholder"
    @blur="onBlur"
    @input="hasError = false"
  />
</template>

<script lang="ts">
import { ref } from "vue";
import isEmpty from "lodash-es/isEmpty";

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
  },
  setup(props, { emit }) {
    const inputField = ref<HTMLInputElement>();
    let hasError = ref(false);

    const onBlur = () => {
      if (props.required && isEmpty(props.value!.trim())) {
        hasError.value = true;
      } else {
        emit("end-editing", props.value);
      }
    };

    return {
      onBlur,
      inputField,
      hasError,
    };
  },
};
</script>
