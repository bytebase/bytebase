<template>
  <div class="relative flex items-start">
    <div class="flex items-center h-5">
      <input
        type="checkbox"
        class="
          h-4
          w-4
          text-accent
          rounded
          disabled:cursor-not-allowed
          border-control-border
          focus:ring-accent
        "
        :disabled="disabled"
        :checked="value"
        @input="
          () => {
            on = !on;
            $emit('toggle', on);
          }
        "
      />
    </div>
    <div v-if="title" class="flex flex-col ml-2 text-sm">
      <label
        class="font-medium"
        :class="disabled ? 'text-gray-400' : 'text-main'"
        >{{ title }}</label
      >
      <template v-if="label">
        <label
          class="mt-1 font-normal"
          :class="disabled ? 'text-gray-400' : 'text-gray-500'"
          >{{ label }}</label
        >
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { ref, watch } from "vue";

export default {
  name: "BBCheckbox",
  props: {
    title: {
      default: "",
      type: String,
    },
    label: {
      default: "",
      type: String,
    },
    value: {
      default: true,
      type: Boolean,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["toggle"],
  setup(props) {
    const on = ref(props.value);
    watch(
      () => props.value,
      (cur) => {
        on.value = cur;
      }
    );
    return { on };
  },
};
</script>
