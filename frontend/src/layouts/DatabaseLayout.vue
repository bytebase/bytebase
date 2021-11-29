<template>
  <router-view
    v-if="dataSourceSlug"
    :database-slug="databaseSlug"
    :data-source-slug="dataSourceSlug"
  />
  <router-view v-else :database-slug="databaseSlug" />
</template>

<script lang="ts">
export default {
  name: "DatabaseLayout",
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    dataSourceSlug: {
      type: String,
    },
  },
};
</script>
