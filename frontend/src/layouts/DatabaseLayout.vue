<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideDatabaseContext :databaseSlug="databaseSlug">
        <!-- This if-lese looks weired because the router-view can be both with or without dataSourceSlug-->
        <router-view
          v-if="dataSourceSlug"
          :databaseSlug="databaseSlug"
          :dataSourceSlug="dataSourceSlug"
        />
        <router-view v-else :databaseSlug="databaseSlug" />
      </ProvideDatabaseContext>
    </template>
    <template #fallback>
      <span>Loading database...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import ProvideDatabaseContext from "../components/ProvideDatabaseContext.vue";

export default {
  name: "DatabaseLayout",
  components: { ProvideDatabaseContext },
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    dataSourceSlug: {
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
