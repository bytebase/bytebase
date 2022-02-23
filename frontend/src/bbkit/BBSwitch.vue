<template>
  <div class="flex items-center">
    <button
      type="button"
      class="relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent disabled:cursor-not-allowed select-none"
      :class="dirtyOn ? 'bg-accent disabled:bg-accent-disabled' : 'bg-gray-200'"
      :disabled="disabled"
      aria-pressed="false"
      @click.prevent="
        () => {
          dirtyOn = !dirtyOn;
          $emit('toggle', dirtyOn);

          dirtyOn = props.value;
        }
      "
    >
      <span class="sr-only">{{ label }}</span>
      <!-- Enabled: "translate-x-5", Not Enabled: "translate-x-0" -->
      <span
        aria-hidden="true"
        class="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
        :class="dirtyOn ? 'translate-x-5' : 'translate-x-0'"
      ></span>
    </button>
    <span
      v-if="label"
      class="ml-2 text-sm font-medium items-center whitespace-nowrap"
      :class="disabled ? 'text-gray-400' : 'text-main'"
    >
      {{ label }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, defineProps, defineEmits, withDefaults } from "vue";

const props = withDefaults(
  defineProps<{
    label?: string;
    value?: boolean;
    disabled?: boolean;
  }>(),
  {
    label: "",
    value: true,
    disabled: false,
  }
);

defineEmits<{
  (event: "toggle", dirtyOn: boolean): void;
}>();

const dirtyOn = ref(props.value);

watch(
  () => props.value,
  (cur) => {
    console.log(dirtyOn, props.value);
    dirtyOn.value = cur;
  }
);
</script>
