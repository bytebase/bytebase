<template>
  <router-view :database-slug="databaseSlug" />
</template>

<script lang="ts">
export default {
  name: "DatabaseLayout",
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
  },
};
</script>
