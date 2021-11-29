<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instance-slug="instanceSlug">
        <!-- This if-lese looks weird because the router-view can be both-->
        <router-view
          v-if="dataSourceSlug"
          :instance-slug="instanceSlug"
          :data-source-slug="dataSourceSlug"
        />
        <router-view v-else :instance-slug="instanceSlug" />
      </ProvideInstanceContext>
    </template>
    <template #fallback>
      <span>Loading instance...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import ProvideInstanceContext from "../components/ProvideInstanceContext.vue";

export default {
  name: "InstanceLayout",
  components: { ProvideInstanceContext },
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
    databaseSlug: {
      type: String,
    },
    dataSourceSlug: {
      type: String,
    },
  },
};
</script>
