<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instance-slug="instanceSlug">
        <router-view :instance-slug="instanceSlug" />
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
      default: "",
    },
  },
};
</script>
