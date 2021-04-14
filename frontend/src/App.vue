<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideContext>
        <router-view />
      </ProvideContext>
    </template>
    <template #fallback>
      <span>Loading...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import { onErrorCaptured } from "vue";
import { useStore } from "vuex";
import ProvideContext from "./components/ProvideContext.vue";
import { isDev } from "./utils";

export default {
  name: "App",
  components: { ProvideContext },
  setup(props, ctx) {
    const store = useStore();

    // Restore previously logged in user if applicable.
    store.dispatch("auth/restoreUser").catch((error: Error) => {
      console.log(error);
    });

    onErrorCaptured((e: any, _, info) => {
      // If e has response, then we assume it's an http error and has already been
      // handled by the axios global handler.
      if (!e.response) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: `Internal error occured`,
          description: isDev() ? e.stack : undefined,
        });
      }
      return true;
    });
  },
};
</script>
