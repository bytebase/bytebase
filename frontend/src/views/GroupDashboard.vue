<template>
  <h1>Group {{ groupSlug }}</h1>
</template>

<script lang="ts">
export default {
  name: "Group Dashboard",
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
