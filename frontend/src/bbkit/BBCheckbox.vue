<template>
  <div class="relative flex items-start">
    <div class="flex items-center h-5">
      <input
        type="checkbox"
        class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
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
    <div v-if="label" class="ml-2 text-sm">
      <label
        class="font-medium"
        :class="disabled ? 'text-gray-400' : 'text-main'"
        >{{ label }}</label
      >
    </div>
  </div>
</template>

<script lang="ts">
import { ref } from "vue";

export default {
  name: "BBCheckbox",
  emits: ["toggle"],
  props: {
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
  setup(props, ctx) {
    const on = ref(props.value);
    return { on };
  },
};
</script>
