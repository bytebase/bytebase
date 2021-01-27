<template>
  <button
    type="button"
    class="btn-primary relative inline-flex items-center px-4 py-2"
  >
    New Project
  </button>
</template>

<script lang="ts">
export default {
  name: "ActionbarGroup",
  components: {},
  props: {
    groupSlug: {
      required: false,
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
