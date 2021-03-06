<template>
  <h1 v-if="error">Failed to load {{ error.stack }}</h1>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense v-else>
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
import { ref, onErrorCaptured } from "vue";
import ProvideContext from "./components/ProvideContext.vue";

export default {
  name: "App",
  components: { ProvideContext },
  setup(props, ctx) {
    const error = ref();

    onErrorCaptured((e) => {
      error.value = e;
      return true;
    });

    return { error };
  },
};
</script>
