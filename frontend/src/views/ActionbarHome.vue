<template>
  <button
    type="button"
    class="btn-primary relative inline-flex items-center px-4 py-2"
  >
    Create Project
  </button>
</template>

<script lang="ts">
export default {
  name: "ActionbarHome",
  components: {},
  props: {
    projectSlug: {
      required: false,
    },
  },
  setup(props, ctx) {},
};
</script>
