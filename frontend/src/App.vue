<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <div>
        <ProvideContext>
          <router-view />
        </ProvideContext>
      </div>
    </template>
    <template #fallback>
      <span>Loading...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import ProvideContext from "./components/ProvideContext.vue";

export default {
  name: "App",
  components: { ProvideContext },
  setup(props, ctx) {},
};
</script>
