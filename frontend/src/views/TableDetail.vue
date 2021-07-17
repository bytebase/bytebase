<template>
  {{ tableName }}
</template>

<script lang="ts">
export default {
  name: "TableDetail",
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    tableName: {
      required: true,
      type: String,
    },
  },
  components: {},
  setup(props, ctx) {},
};
</script>
