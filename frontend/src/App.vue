<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <div>
        <ProvideEnvironment>
          <router-view />
        </ProvideEnvironment>
      </div>
    </template>
    <template #fallback>
      <span>Loading...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import ProvideEnvironment from "./components/ProvideEnvironment.vue";

export default {
  name: "App",
  components: { ProvideEnvironment },
  setup(props, ctx) {},
};
</script>
