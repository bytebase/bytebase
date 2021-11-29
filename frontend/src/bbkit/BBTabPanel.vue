<template>
  <section v-if="active" :aria-hiddem="!active">
    <slot />
  </section>
</template>

<script lang="ts">
export default {
  name: "BBTabPanel",
  props: {
    active: {
      required: true,
      type: Boolean,
    },
  },
};
</script>
