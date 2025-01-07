<template>
  <span
    class="text-white text-[10px] leading-4 px-[6px] py-[2px] rounded-lg select-none bg-control"
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
