<!-- TODO(steven): remove me later please -->
<template>
  <div
    :class="[
      'flex flex-col justify-center items-center p-4',
      border ? 'border-4 border-dashed border-gray-200 rounded-lg' : '',
    ]"
  >
    <div class="text-center flex flex-col justify-center items-center">
      <slot name="image">
        <NEmpty />
      </slot>
    </div>
    <slot />
  </div>
</template>

<script lang="ts" setup>
import { NEmpty } from "naive-ui";
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
