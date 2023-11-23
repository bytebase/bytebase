<template>
  <div class="text-center flex flex-col justify-center items-center">
    <img
      src="@/assets/illustration/no-data.webp"
      class="w-52"
      v-bind="imgAttrs"
    />
    <slot name="default" />
  </div>
</template>

<script setup lang="ts">
import { HTMLAttributes } from "vue";

defineProps<{
  imgAttrs?: HTMLAttributes;
}>();
</script>
