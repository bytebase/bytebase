<template>
  <router-view
    v-if="dataSourceSlug"
    :databaseSlug="databaseSlug"
    :dataSourceSlug="dataSourceSlug"
  />
  <router-view v-else :databaseSlug="databaseSlug" />
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
