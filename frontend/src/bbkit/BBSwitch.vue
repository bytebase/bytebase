<template>
  <div class="flex items-center">
    <button
      type="button"
      class="relative inline-flex flex-shrink-0 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent disabled:cursor-not-allowed select-none"
      :class="[
        `h-${sizes.container}`,
        `w-${sizes.container * 2 - 1}`,
        state.dirtyOn ? 'bg-accent disabled:bg-accent-disabled' : 'bg-gray-200',
      ]"
      :disabled="disabled"
      aria-pressed="false"
      @click.prevent="
        () => {
          state.dirtyOn = !state.dirtyOn;
          $emit('toggle', state.dirtyOn);

          state.dirtyOn = props.value;
        }
      "
    >
      <span class="sr-only">{{ label }}</span>
      <span
        aria-hidden="true"
        class="pointer-events-none inline-block rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
        :class="[
          `w-${sizes.base}`,
          `h-${sizes.base}`,
          state.dirtyOn ? `translate-x-${sizes.base}` : 'translate-x-0',
        ]"
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
import { watch, withDefaults, reactive, computed } from "vue";

export type BBSwitchSize = "small" | "normal";

const props = withDefaults(
  defineProps<{
    size?: BBSwitchSize;
    label?: string;
    value?: boolean;
    disabled?: boolean;
  }>(),
  {
    size: "normal",
    label: "",
    value: true,
    disabled: false,
  }
);

defineEmits<{
  (event: "toggle", dirtyOn: boolean): void;
}>();

const state = reactive({
  dirtyOn: props.value,
});

const sizes = computed(() => {
  const baseSizeMap: Record<BBSwitchSize, number> = {
    normal: 5,
    small: 4,
  };
  const base = baseSizeMap[props.size] ?? baseSizeMap["normal"];
  return {
    base,
    container: base + 1,
  };
});

watch(
  () => props.value,
  (cur) => {
    state.dirtyOn = cur;
  }
);
</script>
