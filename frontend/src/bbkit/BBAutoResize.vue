<template>
  <div ref="control">
    <slot name="main" :resize="resize" />
  </div>
  <slot name="accessory" :resize="resize" />
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
          // Extra 2px is to prevent jiggling upon entering the text
          `${control.value.scrollHeight + 2}px`
        );
      });
    });

    const resize = (el: HTMLTextAreaElement) => {
      el.style.height = "auto";
      el.style.height = `${el.scrollHeight + 2}px`;
    };

    return {
      control,
      resize,
    };
  },
};
</script>
