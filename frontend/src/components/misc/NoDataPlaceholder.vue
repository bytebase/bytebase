<template>
  <div
    class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex flex-col justify-center items-center"
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
import { HTMLAttributes } from "vue";

defineProps<{
  imgAttrs?: HTMLAttributes;
}>();
</script>
