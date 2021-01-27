<template>
  <h1>Group {{ groupSlug }} Setting</h1>
</template>

<script lang="ts">
export default {
  name: "Group Setting",
  components: {},
  props: {
    groupSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
