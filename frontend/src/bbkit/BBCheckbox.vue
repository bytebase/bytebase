<template>
  <div class="relative flex items-start">
    <div class="flex items-center h-5">
      <input
        type="checkbox"
        class="focus:ring-accent h-4 w-4 text-accent border-control-border rounded"
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
      <label class="font-medium text-main">{{ label }}</label>
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
  },
  setup(props, ctx) {
    const on = ref(props.value);
    return { on };
  },
};
</script>
