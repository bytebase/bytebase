<template>
  <section v-if="active" :aria-hiddem="!active" :id="$props.id">
    <slot />
  </section>
</template>

<script lang="ts">
import { computed, reactive } from "vue";

export default {
  name: "BBTabPanel",
  props: {
    active: {
      required: true,
      type: Boolean,
    },
    id: {
      required: true,
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
