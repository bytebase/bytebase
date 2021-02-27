<template>
  <div>
    <slot name="main" :resize="resize" />
  </div>
  <slot name="accessory" :resize="resize" />
</template>

<script lang="ts">
export default {
  name: "ResizeAuto",
  setup(props, ctx) {
    const resize = (el: HTMLTextAreaElement) => {
      el.style.height = "auto";
      // Extra 2px is to prevent jiggling upon entering the text
      el.style.height = `${el.scrollHeight + 2}px`;
    };

    return {
      resize,
    };
  },
};
</script>
