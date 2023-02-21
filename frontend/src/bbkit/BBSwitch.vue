<template>
  <div class="flex items-center">
    <button
      type="button"
      class="relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-accent disabled:cursor-not-allowed select-none"
      :class="
        state.dirtyOn ? 'bg-accent disabled:bg-accent-disabled' : 'bg-gray-200'
      "
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
      <!-- Enabled: "translate-x-5", Not Enabled: "translate-x-0" -->
      <span
        aria-hidden="true"
        class="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200"
        :class="state.dirtyOn ? 'translate-x-5' : 'translate-x-0'"
      ></span>
      <span
        v-if="text"
        aria-hidden="true"
        class="pointer-events-none absolute right-0 top-0 flex items-center justify-center text-[9px] h-5 w-5 transition ease-in-out duration-200"
        :class="
          state.dirtyOn
            ? '-translate-x-5 text-white'
            : 'translate-x-0 text-control'
        "
      >
        {{ state.dirtyOn ? $t("common.on") : $t("common.off") }}
      </span>
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
import { watch, withDefaults, reactive } from "vue";

const props = withDefaults(
  defineProps<{
    text?: boolean;
    label?: string;
    value?: boolean;
    disabled?: boolean;
  }>(),
  {
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

watch(
  () => props.value,
  (cur) => {
    state.dirtyOn = cur;
  }
);
</script>
