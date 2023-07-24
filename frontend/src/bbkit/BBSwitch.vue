<template>
  <div class="flex items-center">
    <button
      type="button"
      class="relative inline-flex flex-shrink-0 transition-colors duration-200 ease-in-out border-2 border-transparent rounded-full cursor-pointer select-none focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent disabled:cursor-not-allowed"
      :class="[
        `w-${sizes.cw}`,
        `h-${sizes.ch}`,
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
        class="inline-block transition duration-200 ease-in-out transform bg-white rounded-full shadow pointer-events-none ring-0"
        :class="[`w-${sizes.base}`, `h-${sizes.base}`]"
        :style="{
          transform: state.dirtyOn ? `translateX(${sizes.base * 0.25}rem)` : '',
        }"
      ></span>
      <span
        v-if="text"
        aria-hidden="true"
        class="absolute top-0 right-0 flex items-center justify-center overflow-hidden transition duration-200 ease-in-out pointer-events-none whitespace-nowrap"
        :class="[
          `w-${sizes.base}`,
          `h-${sizes.base}`,
          state.dirtyOn ? `text-white` : 'text-control',
        ]"
        :style="{
          fontSize: `${sizes.text}px`,
          transform: state.dirtyOn
            ? `translateX(-${sizes.base * 0.25}rem)`
            : '',
        }"
      >
        {{ state.dirtyOn ? $t("common.on") : $t("common.off") }}
      </span>
    </button>
    <span
      v-if="label"
      class="items-center ml-2 text-sm font-medium whitespace-nowrap"
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
    text?: boolean;
    label?: string;
    value?: boolean;
    disabled?: boolean;
  }>(),
  {
    size: "normal",
    text: false,
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
  type Sizes = {
    base: number;
    cw: number; // container width
    ch: number; // container height
    text: number; // unit px
  };
  const sizeMap: Record<BBSwitchSize, Sizes> = {
    normal: { base: 5, cw: 11, ch: 6, text: 10 },
    small: { base: 4, cw: 9, ch: 5, text: 7.5 },
  };
  return sizeMap[props.size] ?? sizeMap["normal"];
});

watch(
  () => props.value,
  (cur) => {
    state.dirtyOn = cur;
  }
);
</script>
