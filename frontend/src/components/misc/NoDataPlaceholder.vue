<template>
  <div
    :class="[
      'flex flex-col justify-center items-center p-4',
      border ? 'border-4 border-dashed border-gray-200 rounded-lg' : '',
    ]"
  >
    <div class="text-center flex flex-col justify-center items-center">
      <slot name="image">
        <img
          src="@/assets/illustration/no-data.webp"
          class="!w-auto max-h-[20vh]"
          v-bind="imgAttrs"
        />
      </slot>
    </div>
    <slot />
  </div>
</template>

<script lang="ts" setup>
import type { HTMLAttributes } from "vue";

withDefaults(
  defineProps<{
    imgAttrs?: HTMLAttributes;
    border?: boolean;
  }>(),
  {
    imgAttrs: undefined,
    border: true,
  }
);
</script>
