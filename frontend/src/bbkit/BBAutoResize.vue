<template>
  <div ref="control">
    <slot :resize="resize" />
  </div>
</template>

<script lang="ts">
import { onMounted, nextTick, ref } from "vue";

export default {
  name: "ResizeAuto",
  setup(props, ctx) {
    const control = ref();

    onMounted(() => {
      nextTick(() => {
        control.value.setAttribute(
          "style",
          "height",
          `${control.value.scrollHeight}px`
        );
      });
    });

    const resize = (e: Event) => {
      const target = e.target as HTMLTextAreaElement;
      target.style.height = "auto";
      target.style.height = `${target.scrollHeight}px`;
    };

    return {
      control,
      resize,
    };
  },
};
</script>
