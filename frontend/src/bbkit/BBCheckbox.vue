<template>
  <div class="relative flex items-start">
    <div class="flex items-center h-5 cursor-pointer">
      <input
        type="checkbox"
        class="h-4 w-4 text-accent rounded cursor-pointer disabled:cursor-not-allowed border-control-border focus:ring-accent"
        :class="[disabled && !value && 'bg-control-bg']"
        :disabled="disabled"
        :checked="value"
        @click.prevent="
          () => {
            on = !on;
            $emit('toggle', on);
            // Prevent dirty state.
            // reference: https://stackoverflow.com/questions/74847997/vue3-controlling-checkbox-checked-state-by-a-component-prop
            on = value;
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

<script lang="ts" setup>
import { ref, watch, withDefaults } from "vue";

const props = withDefaults(
  defineProps<{
    title?: string;
    label?: string;
    value?: boolean;
    disabled?: boolean;
  }>(),
  {
    title: "",
    label: "",
    value: true,
    disabled: false,
  }
);

defineEmits<{
  (event: "toggle", on: boolean): void;
}>();

const on = ref(props.value);
watch(
  () => props.value,
  (cur) => {
    on.value = cur;
  }
);
</script>
