<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instanceSlug="instanceSlug">
        <!-- This if-lese looks weired because the router-view can be both-->
        <router-view
          v-if="dataSourceSlug"
          :instanceSlug="instanceSlug"
          :dataSourceSlug="dataSourceSlug"
        />
        <router-view v-else :instanceSlug="instanceSlug" />
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
    dataSourceSlug: {
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
