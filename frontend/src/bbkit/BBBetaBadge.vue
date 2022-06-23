<template>
  <span
    class="text-white text-xs px-[6px] rounded-lg select-none"
    style="background-color: var(--color-control)"
    :class="
      corner
        ? 'absolute top-0 right-0 translate-x-[50%] translate-y-[-50%]'
        : ''
    "
    >Beta</span
  >
</template>

<script lang="ts" setup>
defineProps({
  corner: {
    type: Boolean,
    default: false,
  },
});
</script>
